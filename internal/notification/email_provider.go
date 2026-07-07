package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

type EmailProvider struct {
	sender MailSender
}

type MailSender interface {
	Send(ctx context.Context, message MailMessage) error
}

type MailMessage struct {
	Address      string
	Host         string
	From         string
	To           []string
	Username     string
	Password     string
	StartTLSMode string
	RequestID    string
	TraceParent  string
	Subject      string
	Body         string
}

func NewEmailProvider(sender MailSender) EmailProvider {
	if sender == nil {
		sender = SMTPMailSender{Timeout: 15 * time.Second}
	}
	return EmailProvider{sender: sender}
}

func (provider EmailProvider) Deliver(ctx context.Context, delivery data.NotificationDelivery) error {
	message, err := parseEmailTarget(delivery.Target)
	if err != nil {
		return err
	}
	message.Subject = notificationTitle(delivery.Title)
	if message.Subject == "" {
		message.Subject = "tictick-hi notification"
	}
	message.RequestID = safeRequestIDHeaderValue(delivery.RequestID)
	message.TraceParent = safeTraceParentHeaderValue(delivery.TraceParent)
	message.Body = notificationText(delivery.Title, delivery.Body)
	if err := provider.sender.Send(ctx, message); err != nil {
		return fmt.Errorf("deliver email notification: %s", redactedError(err.Error(), message.Password))
	}
	return nil
}

func parseEmailTarget(target string) (MailMessage, error) {
	parsed, values, err := parseTargetURL(target, "smtp")
	if err != nil {
		return MailMessage{}, err
	}
	if parsed.Host == "" {
		return MailMessage{}, fmt.Errorf("smtp target host is required")
	}
	address := parsed.Host
	if parsed.Port() == "" {
		address = net.JoinHostPort(parsed.Hostname(), "587")
	}
	from, err := requiredParam(values, "from")
	if err != nil {
		return MailMessage{}, err
	}
	toValue, err := requiredParam(values, "to")
	if err != nil {
		return MailMessage{}, err
	}
	recipients, err := splitRecipients(toValue)
	if err != nil {
		return MailMessage{}, err
	}
	_, username, err := optionalEnv(values, "username_env")
	if err != nil {
		return MailMessage{}, err
	}
	_, password, err := optionalEnv(values, "password_env")
	if err != nil {
		return MailMessage{}, err
	}
	if password != "" && username == "" {
		return MailMessage{}, fmt.Errorf("username_env is required when password_env is set")
	}
	startTLSMode := strings.TrimSpace(values.Get("starttls"))
	if startTLSMode == "" {
		startTLSMode = "opportunistic"
	}
	if startTLSMode != "opportunistic" && startTLSMode != "required" && startTLSMode != "disabled" {
		return MailMessage{}, fmt.Errorf("starttls must be opportunistic, required or disabled")
	}
	return MailMessage{
		Address:      address,
		Host:         parsed.Hostname(),
		From:         from,
		To:           recipients,
		Username:     username,
		Password:     password,
		StartTLSMode: startTLSMode,
	}, nil
}

func validateEmailTargetSyntax(target string) error {
	parsed, values, err := parseTargetURL(target, "smtp")
	if err != nil {
		return err
	}
	if parsed.Host == "" {
		return fmt.Errorf("smtp target host is required")
	}
	if _, err := requiredParam(values, "from"); err != nil {
		return err
	}
	toValue, err := requiredParam(values, "to")
	if err != nil {
		return err
	}
	if _, err := splitRecipients(toValue); err != nil {
		return err
	}
	usernameEnv, err := optionalEnvReference(values, "username_env")
	if err != nil {
		return err
	}
	passwordEnv, err := optionalEnvReference(values, "password_env")
	if err != nil {
		return err
	}
	if passwordEnv != "" && usernameEnv == "" {
		return fmt.Errorf("username_env is required when password_env is set")
	}
	startTLSMode := strings.TrimSpace(values.Get("starttls"))
	if startTLSMode == "" {
		startTLSMode = "opportunistic"
	}
	if startTLSMode != "opportunistic" && startTLSMode != "required" && startTLSMode != "disabled" {
		return fmt.Errorf("starttls must be opportunistic, required or disabled")
	}
	return nil
}

type SMTPMailSender struct {
	Timeout time.Duration
}

func (sender SMTPMailSender) Send(ctx context.Context, message MailMessage) error {
	timeout := sender.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", message.Address)
	if err != nil {
		return fmt.Errorf("dial smtp server: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	client, err := smtp.NewClient(conn, message.Host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if err := sender.startTLS(client, message); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if message.Username != "" {
		auth := smtp.PlainAuth("", message.Username, message.Password, message.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := client.Mail(message.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	for _, recipient := range message.To {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp recipient: %w", err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write([]byte(formatEmail(message))); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write email body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close email body: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

func (sender SMTPMailSender) startTLS(client *smtp.Client, message MailMessage) error {
	if message.StartTLSMode == "disabled" {
		return nil
	}
	if ok, _ := client.Extension("STARTTLS"); !ok {
		if message.StartTLSMode == "required" {
			return fmt.Errorf("smtp server does not support STARTTLS")
		}
		return nil
	}
	config := &tls.Config{ServerName: message.Host, MinVersion: tls.VersionTLS12}
	if err := client.StartTLS(config); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}
	return nil
}

func formatEmail(message MailMessage) string {
	headers := textproto.MIMEHeader{}
	headers.Set("From", message.From)
	headers.Set("To", strings.Join(message.To, ", "))
	headers.Set("Subject", message.Subject)
	headers.Set("MIME-Version", "1.0")
	headers.Set("Content-Type", "text/plain; charset=UTF-8")
	if message.RequestID != "" {
		headers.Set(outboundRequestIDHeader, message.RequestID)
	}
	if message.TraceParent != "" {
		headers.Set(outboundTraceParentHeader, message.TraceParent)
	}

	var builder strings.Builder
	for key, values := range headers {
		for _, value := range values {
			builder.WriteString(key)
			builder.WriteString(": ")
			builder.WriteString(strings.ReplaceAll(value, "\n", " "))
			builder.WriteString("\r\n")
		}
	}
	builder.WriteString("\r\n")
	builder.WriteString(strings.ReplaceAll(message.Body, "\n", "\r\n"))
	builder.WriteString("\r\n")
	return builder.String()
}
