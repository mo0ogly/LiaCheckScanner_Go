#!/bin/bash

echo "🚀 Setup GitHub pour LiaCheckScanner Go"
echo "======================================"

echo ""
echo "ÉTAPE 1: Créer le repository sur GitHub"
echo "---------------------------------------"
echo "1. Allez sur: https://github.com/mo0ogly"
echo "2. Cliquez sur le bouton vert 'New repository'"
echo "3. Repository name: LiaCheckScanner_Go"
echo "4. Description: Scanner IP extractor and RDAP enrichment tool"
echo "5. ✅ Public"
echo "6. ✅ Add a README file"
echo "7. ✅ Add .gitignore (choose Go)"
echo "8. ✅ Choose a license (MIT)"
echo "9. Cliquez 'Create repository'"
echo ""
echo "ATTENDEZ! N'appuyez sur Entrée qu'APRÈS avoir créé le repo sur GitHub..."
read -p "Repository créé sur GitHub ? (Entrée pour continuer): "

echo ""
echo "ÉTAPE 2: Initialisation Git locale"
echo "-----------------------------------"

# Sortir du git parent et créer un nouveau git
rm -rf .git
git init
git config user.name "mo0ogly"
git config user.email "mo0ogly@proton.me"

echo "✅ Git initialisé"

echo ""
echo "ÉTAPE 3: Premier commit"
echo "----------------------"

git add .
git commit -m "Initial commit: LiaCheckScanner Go

Simple tool for:
- IP extraction from internet-scanners repository
- RDAP enrichment from 5 major registries  
- GUI interface with Fyne framework
- CSV export capabilities

Built with Go, simple and functional."

echo "✅ Commit créé"

echo ""
echo "ÉTAPE 4: Connexion à GitHub"
echo "---------------------------"

git branch -M main
git remote add origin https://github.com/mo0ogly/LiaCheckScanner_Go.git

echo ""
echo "ÉTAPE 5: Push vers GitHub"
echo "------------------------"

git push -u origin main

echo ""
echo "🎉 TERMINÉ!"
echo "==========="
echo "Votre repository est maintenant disponible sur:"
echo "👉 https://github.com/mo0ogly/LiaCheckScanner_Go"
echo ""
echo "📝 N'oubliez pas d'ajouter les topics dans GitHub:"
echo "   Settings > General > Topics: go, rdap, ip-scanner, whois, fyne-gui"
