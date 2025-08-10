-- +goose Up
CREATE TABLE moves(
  id SERIAL PRIMARY KEY,
  board JSON NOT NULL,
  move TEXT NOT NULL,
  white_time INT NOT NULL,
  black_time INT NOT NULL,
  match_id INT NOT NULL,
  FOREIGN KEY (match_id)
  REFERENCES matches(ID)
  ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE moves;