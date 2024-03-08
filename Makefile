# Build stage
build:
	@echo "Creating docker compose..."
	docker compose create
	@echo "Building server app..."
	go build -o server_app ./app1/main.go 
	@echo "Building client app..."
	go build -o client_app ./app2/main.go 
	@echo "Build stage completed."

setup:
	@echo "Setting up docker compose..."
	docker compose up -d
	@echo "Setting up server app..."
	./server_app & echo $$! > server_app.pid
	@echo "Setup stage completed."

run:
	@echo "Running client app..."
	./client_app 
	@echo "Run stage completed."
	
clean:
	@echo "Cleaning up..."
	docker compose down
	@echo "Cleaning up server app..."
	kill `cat server_app.pid`
	rm -f server_app server_app.pid
	@echo "Cleaning up client app..."
	rm -f client_app
	@echo "Clean stage completed."