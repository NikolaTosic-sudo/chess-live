-- +goose Up
CREATE TABLE matches(
  id SERIAL PRIMARY KEY,
  white TEXT NOT NULL,
  black TEXT NOT NULL,
  full_time INT NOT NULL,
  is_online BOOLEAN DEFAULT FALSE NOT NULL,
  user_id UUID NOT NULL,
  FOREIGN KEY (user_id)
  REFERENCES users(ID)
  ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE matches;