ALTER TABLE exchange_accounts
  ADD CONSTRAINT exchange_accounts_trimmed_required_text_check
    CHECK (
      btrim(exchange) <> '' AND
      btrim(alias) <> '' AND
      btrim(encrypted_api_key) <> '' AND
      btrim(encrypted_api_secret) <> ''
    ) NOT VALID;
