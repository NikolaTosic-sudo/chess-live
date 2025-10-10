-- +goose Up
CREATE TABLE matches_users(
  match_id INTEGER NOT NULL,
  user_id UUID NOT NULL
);

-- +goose Down
DROP TABLE matches_users;
