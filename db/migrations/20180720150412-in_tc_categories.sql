
-- +migrate Up notransaction
ALTER TYPE  "domain" ADD VALUE 'in';
ALTER TYPE  "domain" ADD VALUE 'tc';

-- +migrate Down
