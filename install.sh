#!/bin/bash

# LiaCheckScanner_Go - Script d'installation Go
# Owner: LIA - mo0ogly@proton.me

set -e

# Couleurs pour l'affichage
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Variables
APP_NAME="LiaCheckScanner_Go"
VERSION="1.0.0"
OWNER="LIA - mo0ogly@proton.me"
REPO_URL="https://github.com/lia/liacheckscanner_go"

echo -e "${BLUE}🔍 $APP_NAME - Installation Go${NC}"
echo -e "${YELLOW}Owner: $OWNER${NC}"
echo -e "${YELLOW}Version: $VERSION${NC}"
echo ""

# Fonction pour afficher les messages d'erreur
error_exit() {
    echo -e "${RED}❌ Erreur: $1${NC}" >&2
    exit 1
}

# Fonction pour afficher les messages de succès
success_msg() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Fonction pour afficher les messages d'information
info_msg() {
    echo -e "${BLUE}ℹ️ $1${NC}"
}

# Fonction pour afficher les messages d'avertissement
warning_msg() {
    echo -e "${YELLOW}⚠️ $1${NC}"
}

# Vérifier si Go est installé
check_go() {
    info_msg "Vérification de Go..."
    
    if ! command -v go &> /dev/null; then
        error_exit "Go n'est pas installé. Veuillez installer Go 1.21+ depuis https://golang.org/dl/"
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    GO_MAJOR=$(echo $GO_VERSION | cut -d. -f1)
    GO_MINOR=$(echo $GO_VERSION | cut -d. -f2)
    
    if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 21 ]); then
        error_exit "Go $GO_VERSION détecté. Go 1.21+ est requis."
    fi
    
    success_msg "Go $GO_VERSION détecté"
}

# Vérifier les prérequis système
check_prerequisites() {
    info_msg "Vérification des prérequis..."
    
    # Vérifier Git
    if ! command -v git &> /dev/null; then
        error_exit "Git n'est pas installé. Veuillez installer Git."
    fi
    success_msg "Git détecté"
    
    # Vérifier make
    if ! command -v make &> /dev/null; then
        warning_msg "Make non détecté. Certaines fonctionnalités peuvent ne pas être disponibles."
    else
        success_msg "Make détecté"
    fi
    
    # Vérifier l'espace disque
    AVAILABLE_SPACE=$(df . | awk 'NR==2 {print $4}')
    if [ "$AVAILABLE_SPACE" -lt 1048576 ]; then # 1GB en KB
        warning_msg "Espace disque faible. Au moins 1GB recommandé."
    else
        success_msg "Espace disque suffisant"
    fi
}

# Créer la structure des dossiers
create_directories() {
    info_msg "Création de la structure des dossiers..."
    
    DIRS=("logs" "results" "data" "config" "assets/icons" "build")
    
    for dir in "${DIRS[@]}"; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
            success_msg "Dossier créé: $dir"
        else
            info_msg "Dossier existant: $dir"
        fi
    done
}

# Télécharger les dépendances
download_dependencies() {
    info_msg "Téléchargement des dépendances Go..."
    
    if [ -f "go.mod" ]; then
        go mod download
        go mod tidy
        success_msg "Dépendances téléchargées"
    else
        error_exit "Fichier go.mod non trouvé"
    fi
}

# Compiler l'application
build_application() {
    info_msg "Compilation de l'application..."
    
    # Compilation standard
    go build -ldflags="-s -w -X main.Version=$VERSION" -o build/liacheckscanner ./cmd/liacheckscanner
    
    if [ $? -eq 0 ]; then
        success_msg "Application compilée: build/liacheckscanner"
    else
        error_exit "Erreur lors de la compilation"
    fi
}

# Créer les scripts de lancement
create_launch_scripts() {
    info_msg "Création des scripts de lancement..."
    
    # Script de lancement principal
    cat > run.sh << 'EOF'
#!/bin/bash
# LiaCheckScanner - Script de lancement
# Owner: LIA - mo0ogly@proton.me

# Vérifier si l'exécutable existe
if [ -f "./build/liacheckscanner" ]; then
    ./build/liacheckscanner
elif [ -f "./liacheckscanner" ]; then
    ./liacheckscanner
else
    echo "❌ Exécutable non trouvé. Lancement avec go run..."
    go run ./cmd/liacheckscanner
fi
EOF

    chmod +x run.sh
    success_msg "Script de lancement créé: run.sh"
    
    # Script de développement
    cat > dev.sh << 'EOF'
#!/bin/bash
# LiaCheckScanner - Mode développement
# Owner: LIA - mo0ogly@proton.me

echo "🔥 Mode développement..."
go run ./cmd/liacheckscanner
EOF

    chmod +x dev.sh
    success_msg "Script de développement créé: dev.sh"
}

# Installer les outils de développement (optionnel)
install_dev_tools() {
    if [ "$1" = "--dev" ]; then
        info_msg "Installation des outils de développement..."
        
        # Air pour le hot reload
        if ! command -v air &> /dev/null; then
            go install github.com/cosmtrek/air@latest
            success_msg "Air installé pour le hot reload"
        fi
        
        # golangci-lint pour le linting
        if ! command -v golangci-lint &> /dev/null; then
            curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
            success_msg "golangci-lint installé"
        fi
        
        # Delve pour le debugging
        if ! command -v dlv &> /dev/null; then
            go install github.com/go-delve/delve/cmd/dlv@latest
            success_msg "Delve installé pour le debugging"
        fi
    fi
}

# Créer la configuration par défaut
create_default_config() {
    info_msg "Création de la configuration par défaut..."
    
    if [ ! -f "config/config.json" ]; then
        cat > config/config.json << 'EOF'
{
  "app_name": "LiaCheckScanner",
  "version": "1.0.0",
  "owner": "LIA - mo0ogly@proton.me",
  "theme": "dark",
  "language": "fr",
  "log_level": "INFO",
  "max_log_size": 10,
  "log_backups": 5,
  "database": {
    "repo_url": "https://github.com/six2dez/reconftw",
    "local_path": "./data/repository",
    "results_dir": "./results",
    "logs_dir": "./logs",
    "api_key": "",
    "enable_api": false,
    "api_throttle": 1.0,
    "auto_update": false,
    "update_interval": 24
  }
}
EOF
        success_msg "Configuration par défaut créée: config/config.json"
    else
        info_msg "Configuration existante: config/config.json"
    fi
}

# Afficher les informations de fin
show_completion_info() {
    echo ""
    echo -e "${GREEN}🎉 Installation terminée avec succès !${NC}"
    echo ""
    echo -e "${BLUE}📋 Informations:${NC}"
    echo -e "  • Application: $APP_NAME v$VERSION"
    echo -e "  • Owner: $OWNER"
    echo -e "  • Exécutable: ./build/liacheckscanner"
    echo ""
    echo -e "${BLUE}🚀 Lancement:${NC}"
    echo -e "  • Normal: ./run.sh"
    echo -e "  • Développement: ./dev.sh"
    echo -e "  • Direct: go run ./cmd/liacheckscanner"
    echo ""
    echo -e "${BLUE}🔧 Commandes utiles:${NC}"
    echo -e "  • Aide: make help"
    echo -e "  • Tests: make test"
    echo -e "  • Build: make build"
    echo -e "  • Clean: make clean"
    echo ""
    echo -e "${YELLOW}📚 Documentation: README.md${NC}"
}

# Fonction principale
main() {
    echo -e "${BLUE}🔍 Début de l'installation de $APP_NAME...${NC}"
    echo ""
    
    check_go
    check_prerequisites
    create_directories
    download_dependencies
    build_application
    create_launch_scripts
    install_dev_tools "$1"
    create_default_config
    show_completion_info
}

# Gestion des arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --dev     Installer les outils de développement"
        echo "  --help    Afficher cette aide"
        echo ""
        echo "Owner: $OWNER"
        exit 0
        ;;
    --dev)
        main "$1"
        ;;
    "")
        main
        ;;
    *)
        error_exit "Option inconnue: $1. Utilisez --help pour l'aide."
        ;;
esac 