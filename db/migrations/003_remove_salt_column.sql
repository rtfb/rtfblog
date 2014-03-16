-- +goose Up
alter table author drop column salt;

-- +goose Down
alter table author add column salt text;
