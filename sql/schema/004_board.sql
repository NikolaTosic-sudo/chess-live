-- +goose Up
CREATE TABLE board(
  id INT SERIAL PRIMARY KEY,

  whiteTime INT NOT NULL,
  blackTime INT NOT NULL,
  matchId INT NOT NULL,
  FOREIGN KEY (matchId)
  REFERENCE matches(id)
  ON CASCADE DELETE,
  created_at TIMESTAMP NOT NULL
);

-- +goose Down