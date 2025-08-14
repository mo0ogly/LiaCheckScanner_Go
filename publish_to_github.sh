#!/bin/bash

echo "🚀 Publication automatique de LiaCheckScanner Go sur GitHub"
echo "=================================================="

# 1. Créer le repository sur GitHub
echo "1. Allez sur https://github.com/mo0ogly"
echo "2. Cliquez sur 'New repository'"
echo "3. Repository name: LiaCheckScanner_Go"
echo "4. Description: Scanner IP extractor and RDAP enrichment tool"
echo "5. ✅ Public, ✅ README, ✅ .gitignore (Go), ✅ MIT License"
echo "6. Create repository"
echo ""
echo "Appuyez sur Entrée quand c'est fait..."
read

# 2. Configuration Git
echo "🔧 Configuration Git locale..."
git init
git config user.name "mo0ogly"
git config user.email "mo0ogly@proton.me"

# 3. Ajout du remote
echo "🔗 Ajout du remote GitHub..."
git remote add origin https://github.com/mo0ogly/LiaCheckScanner_Go.git

# 4. Premier commit
echo "📝 Préparation du commit initial..."
git add .
git commit -m "Initial commit: LiaCheckScanner Go

Simple tool for:
- IP extraction from internet-scanners repository  
- RDAP enrichment from 5 major registries
- GUI interface with Fyne framework
- CSV export capabilities

Built with Go, no pretensions, just works."

# 5. Push vers GitHub
echo "🚀 Push vers GitHub..."
git branch -M main
git push -u origin main

echo ""
echo "✅ TERMINÉ! Votre repository est maintenant sur GitHub:"
echo "   https://github.com/mo0ogly/LiaCheckScanner_Go"
echo ""
echo "🎯 Topics à ajouter dans Settings:"
echo "   go, rdap, ip-scanner, whois, netfilter, fyne-gui"
