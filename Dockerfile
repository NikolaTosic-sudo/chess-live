FROM debian:stable-slim
RUN apt-get update && apt-get install -y ca-certificates && apt-get install -y postgresql postgresql-contrib --fix-missing
RUN curl -sS https://webi.sh/golang | sh; \ source ~/.config/envman/PATH.env
ADD chess-live /usr/bin/chess-live
COPY assets/ assets/
RUN goose postgres "user=nikolatosic dbname=chess sslmode=disable" up
EXPOSE 8080
CMD ["/usr/bin/chess-live"]
