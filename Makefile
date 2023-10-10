
up:
	migrate -source file://migrations -database postgres://docker:docker@localhost:5432/tongate_dev?sslmode=disable up