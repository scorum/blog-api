
-- +migrate Up  notransaction
ALTER TYPE  "plagiarism_status" ADD VALUE 'invalid_text_len';
-- +migrate Down
