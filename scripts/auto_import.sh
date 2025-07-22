#!/bin/bash
# Script para automação da importação de dados de criptomoedas
# Executa o script Python de importação periodicamente

# Ativa o ambiente virtual Python
source venv/bin/activate

# Executa o script de importação
python scripts/import_price_history.py

# Sugestão de uso via cron (exemplo para rodar a cada hora):
# 0 * * * * /home/felipe-macedo/projects/go-priceguard-api/scripts/auto_import.sh >> /home/felipe-macedo/projects/go-priceguard-api/scripts/auto_import.log 2>&1

# Para agendar via cron:
# 1. Edite o crontab: crontab -e
# 2. Adicione a linha acima (ajuste o caminho conforme necessário)

# Para rodar manualmente:
# bash scripts/auto_import.sh
