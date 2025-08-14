# LiaCheckScanner_Go - Makefile
# Owner: LIA - mo0ogly@proton.me

.PHONY: help build clean test install run dev build-all build-linux build-windows build-darwin

# Variables
APP_NAME = liacheckscanner
VERSION = 1.0.0
OWNER = LIA - mo0ogly@proton.me
BUILD_DIR = build
MAIN_PATH = ./cmd/liacheckscanner

# Couleurs pour l'affichage
GREEN = \033[32m
YELLOW = \033[33m
RED = \033[31m
BLUE = \033[34m
RESET = \033[0m

help: ## Afficher cette aide
	@echo "$(BLUE)🔍 LiaCheckScanner_Go - Makefile$(RESET)"
	@echo "$(YELLOW)Owner: $(OWNER)$(RESET)"
	@echo ""
	@echo "$(GREEN)Commandes disponibles:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(RESET) %s\n", $$1, $$2}'

build: ## Compiler l'application
	@echo "$(GREEN)🔨 Compilation de $(APP_NAME)...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✅ Compilation terminée: $(BUILD_DIR)/$(APP_NAME)$(RESET)"

clean: ## Nettoyer les fichiers de build
	@echo "$(YELLOW)🧹 Nettoyage...$(RESET)"
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "$(GREEN)✅ Nettoyage terminé$(RESET)"

test: ## Lancer les tests
	@echo "$(BLUE)🧪 Lancement des tests...$(RESET)"
	go test -v ./...
	@echo "$(GREEN)✅ Tests terminés$(RESET)"

test-coverage: ## Lancer les tests avec couverture
	@echo "$(BLUE)🧪 Tests avec couverture...$(RESET)"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Rapport de couverture généré: coverage.html$(RESET)"

install: ## Installer l'application
	@echo "$(GREEN)📦 Installation...$(RESET)"
	go install $(MAIN_PATH)
	@echo "$(GREEN)✅ Installation terminée$(RESET)"

run: ## Lancer l'application
	@echo "$(BLUE)🚀 Lancement de $(APP_NAME)...$(RESET)"
	go run $(MAIN_PATH)

dev: ## Mode développement avec hot reload
	@echo "$(BLUE)🔥 Mode développement...$(RESET)"
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "$(YELLOW)⚠️ Air non installé, lancement normal...$(RESET)"; \
		go run $(MAIN_PATH); \
	fi

build-all: build-linux build-windows build-darwin ## Compiler pour toutes les plateformes

build-linux: ## Compiler pour Linux
	@echo "$(GREEN)🐧 Compilation pour Linux...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(MAIN_PATH)
	@echo "$(GREEN)✅ Compilation Linux terminée$(RESET)"

build-windows: ## Compiler pour Windows
	@echo "$(GREEN)🪟 Compilation pour Windows...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "$(GREEN)✅ Compilation Windows terminée$(RESET)"

build-darwin: ## Compiler pour macOS
	@echo "$(GREEN)🍎 Compilation pour macOS...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "$(GREEN)✅ Compilation macOS terminée$(RESET)"

deps: ## Télécharger les dépendances
	@echo "$(BLUE)📥 Téléchargement des dépendances...$(RESET)"
	go mod download
	go mod tidy
	@echo "$(GREEN)✅ Dépendances téléchargées$(RESET)"

fmt: ## Formater le code
	@echo "$(BLUE)🎨 Formatage du code...$(RESET)"
	go fmt ./...
	@echo "$(GREEN)✅ Code formaté$(RESET)"

vet: ## Vérifier le code
	@echo "$(BLUE)🔍 Vérification du code...$(RESET)"
	go vet ./...
	@echo "$(GREEN)✅ Code vérifié$(RESET)"

lint: ## Linter le code
	@echo "$(BLUE)🔍 Linting du code...$(RESET)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)⚠️ golangci-lint non installé$(RESET)"; \
	fi

bench: ## Benchmarks
	@echo "$(BLUE)⚡ Benchmarks...$(RESET)"
	go test -bench=. ./...
	@echo "$(GREEN)✅ Benchmarks terminés$(RESET)"

release: clean build-all ## Créer une release
	@echo "$(GREEN)🎉 Release créée dans $(BUILD_DIR)/$(RESET)"
	@ls -la $(BUILD_DIR)/

setup: ## Configuration initiale
	@echo "$(BLUE)⚙️ Configuration initiale...$(RESET)"
	@mkdir -p logs results data config assets/icons
	@echo "$(GREEN)✅ Configuration terminée$(RESET)"

docker-build: ## Build Docker
	@echo "$(BLUE)🐳 Build Docker...$(RESET)"
	docker build -t $(APP_NAME):$(VERSION) .
	@echo "$(GREEN)✅ Docker build terminé$(RESET)"

docker-run: ## Run Docker
	@echo "$(BLUE)🐳 Run Docker...$(RESET)"
	docker run -it --rm $(APP_NAME):$(VERSION) 