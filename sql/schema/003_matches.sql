-- +goose Up
CREATE TABLE matches(
  id SERIAL PRIMARY KEY,
  white TEXT NOT NULL,
  black TEXT NOT NULL,
  full_time INT NOT NULL,
  is_online BOOLEAN DEFAULT FALSE NOT NULL,
  result TEXT NOT NULL,
  ended BOOLEAN DEFAULT FALSE NOT NULL,
  created_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE matches;
