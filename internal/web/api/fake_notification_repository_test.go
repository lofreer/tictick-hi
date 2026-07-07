package api

import (
	"context"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (repository *fakeRepository) ListNotifications(context.Context) ([]data.Notification, error) {
	return append([]data.Notification(nil), repository.notifications...), nil
}

func (repository *fakeRepository) RetryNotification(_ context.Context, id string) (data.Notification, error) {
	for index := range repository.notifications {
		if repository.notifications[index].ID == id {
			repository.notifications[index].Status = "pending"
			repository.notifications[index].Error = ""
			return repository.notifications[index], nil
		}
	}
	return data.Notification{}, data.ErrNotFound
}

func (repository *fakeRepository) ListNotificationChannels(context.Context) ([]data.NotificationChannel, error) {
	return append([]data.NotificationChannel(nil), repository.channels...), nil
}

func (repository *fakeRepository) CreateNotificationChannel(
	_ context.Context,
	request data.CreateNotificationChannel,
) (data.NotificationChannel, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	channel := data.NotificationChannel{
		ID:        "nc_1",
		Name:      request.Name,
		Provider:  request.Provider,
		Target:    request.Target,
		Enabled:   request.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.channels = append(repository.channels, channel)
	return channel, nil
}

func (repository *fakeRepository) UpdateNotificationChannel(
	_ context.Context,
	id string,
	request data.CreateNotificationChannel,
) (data.NotificationChannel, error) {
	for index := range repository.channels {
		if repository.channels[index].ID == id {
			repository.channels[index].Name = request.Name
			repository.channels[index].Provider = request.Provider
			repository.channels[index].Target = request.Target
			repository.channels[index].Enabled = request.Enabled
			repository.channels[index].UpdatedAt = time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC)
			return repository.channels[index], nil
		}
	}
	return data.NotificationChannel{}, data.ErrNotFound
}

func (repository *fakeRepository) DeleteNotificationChannel(
	_ context.Context,
	id string,
) (data.NotificationChannel, error) {
	for index := range repository.channels {
		if repository.channels[index].ID == id {
			channel := repository.channels[index]
			repository.channels = append(repository.channels[:index], repository.channels[index+1:]...)
			return channel, nil
		}
	}
	return data.NotificationChannel{}, data.ErrNotFound
}

func (repository *fakeRepository) SetNotificationChannelEnabled(
	_ context.Context,
	id string,
	enabled bool,
) (data.NotificationChannel, error) {
	for index := range repository.channels {
		if repository.channels[index].ID == id {
			repository.channels[index].Enabled = enabled
			repository.channels[index].UpdatedAt = time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
			return repository.channels[index], nil
		}
	}
	return data.NotificationChannel{}, data.ErrNotFound
}
