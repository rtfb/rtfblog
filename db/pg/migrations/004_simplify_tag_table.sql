-- +goose Up
alter table tag drop column name;
alter table tag rename column url to tag;

-- +goose Down
alter table tag add column name text;
alter table tag rename column tag to url;
