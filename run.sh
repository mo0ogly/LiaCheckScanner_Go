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
