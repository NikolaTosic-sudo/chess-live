-- +goose Up
CREATE TABLE board(
  id SERIAL PRIMARY KEY,
  board JSON NOT NULL,
  move TEXT NOT NULL,
  whiteTime INT NOT NULL,
  blackTime INT NOT NULL,
  match_id INT NOT NULL,
  FOREIGN KEY (match_id)
  REFERENCES matches(ID)
  ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE board;