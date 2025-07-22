import psycopg2
from datetime import datetime
import requests
import time

# Configuração do banco
conn = psycopg2.connect(
    dbname='priceguard',
    user='postgres',
    password='password',  # Corrigido para o valor do docker-compose
    host='localhost',
    port=5432
)
cursor = conn.cursor()

timeframes = {
    '1m': '1m',
    '5m': '5m',
    '15m': '15m',
    '1h': '1h',
    '4h': '4h',
    '1d': '1d'
}

# Busca os símbolos das top 100 moedas por market cap
resp = requests.get('https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1')
coins = resp.json()
symbols = []
for coin in coins:
    binance_symbol = coin['symbol'].upper() + 'USDT'
    symbols.append(binance_symbol)

# Função para buscar candles da Binance
BINANCE_URL = 'https://api.binance.com/api/v3/klines'
def fetch_binance_candles(symbol, interval, limit=200):
    params = {
        'symbol': symbol,
        'interval': interval,
        'limit': limit
    }
    r = requests.get(BINANCE_URL, params=params)
    if r.status_code == 200:
        return r.json()
    return []

# Passo 1: Atualiza/insere metadados das top 100 moedas no banco
for coin in coins:
    binance_symbol = coin['symbol'].upper() + 'USDT'
    # Insere ou atualiza na tabela cryptocurrencies
    cursor.execute(
        """
        INSERT INTO cryptocurrencies (symbol, name, market_type, image_url, active)
        VALUES (%s, %s, %s, %s, true)
        ON CONFLICT (symbol) DO UPDATE SET name = EXCLUDED.name, image_url = EXCLUDED.image_url, active = true
        """,
        (binance_symbol, coin['name'], 'Spot', coin.get('image', ''))
    )
    symbols.append(binance_symbol)

# Passo 2: Importa candles reais da Binance para cada símbolo/timeframe
for symbol in symbols:
    for tf_name, tf_binance in timeframes.items():
        print(f'Importando candles reais para {symbol} - {tf_name}...')
        candles = fetch_binance_candles(symbol, tf_binance)
        for candle in candles:
            open_time = datetime.utcfromtimestamp(candle[0] / 1000)
            open_price = float(candle[1])
            high_price = float(candle[2])
            low_price = float(candle[3])
            close_price = float(candle[4])
            volume = float(candle[5])
            cursor.execute(
                """
                INSERT INTO price_history (symbol, timeframe, open_price, high_price, low_price, close_price, volume, timestamp, created_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT DO NOTHING
                """,
                (symbol, tf_name, open_price, high_price, low_price, close_price, volume, open_time, open_time)
            )
        time.sleep(0.2)  # Evita rate limit da Binance

conn.commit()
cursor.close()
conn.close()
print('Importação concluída!')
