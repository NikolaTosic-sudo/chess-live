-- +goose Up
CREATE TABLE matches(
  id INT SERIAL PRIMARY KEY,
  white TEXT NOT NULL,
  black TEXT NOT NULL,
  timer INT NOT NULL,
  userId UUID NOT NULL,
  FOREIGN KEY (user_id)
  REFERENCES users(ID)
  ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL,
);

-- +goose Down
DROP TABLE matches;