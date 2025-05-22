
-- +migrate Up notransaction
ALTER TYPE  "notification_type" ADD VALUE 'post_uniqueness_checked';

-- +migrate Down
