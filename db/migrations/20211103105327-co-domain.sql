
-- +migrate Up notransaction
ALTER TYPE  "domain" ADD VALUE IF NOT EXISTS 'co';
-- +migrate Down
