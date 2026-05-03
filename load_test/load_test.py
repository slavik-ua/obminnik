import requests
import time
import random
import threading
from concurrent.futures import ThreadPoolExecutor

# --- Configuration ---
BASE_URL = "http://127.0.0.1:8000"
NUM_TRADERS = 10
ORDERS_PER_SEC = 2    # Speed per trader
PRICE_SPREAD = 100    # Spread around mid price
DECIMALS = 10**8      # 1e8 scaling

# --- Shared Market State ---
# This ensures all traders fight over the same price levels
class GlobalMarket:
    def __init__(self):
        self.mid_price = 20000.0  # $20k BTC
        self.lock = threading.Lock()

    def update(self):
        with self.lock:
            # Random walk: move price by small amount
            self.mid_price += random.uniform(-10, 10)
            self.mid_price = max(1000, min(100000, self.mid_price))
            return int(self.mid_price)

market = GlobalMarket()

def trader_logic(trader_id):
    """ Main loop for a single virtual trader """
    email = f"bot_{trader_id}_{random.randint(1000, 9999)}@exchange.com"
    password = "password123"
    session = requests.Session() # Uses Keep-Alive for high performance

    # 1. Setup Auth & Initial Funds
    try:
        session.post(f"{BASE_URL}/register", json={"email": email, "password": password})
        login_res = session.post(f"{BASE_URL}/login", json={"email": email, "password": password})
        token = login_res.json().get("token")
        headers = {"Authorization": f"Bearer {token}"}
        
        # Give bots plenty of money to trade
        session.post(f"{BASE_URL}/deposit", json={"asset": "USD", "amount": 1000000 * DECIMALS}, headers=headers)
        session.post(f"{BASE_URL}/deposit", json={"asset": "BTC", "amount": 100 * DECIMALS}, headers=headers)
    except Exception as e:
        print(f"Trader {trader_id} failed setup: {e}")
        return

    print(f"Trader {trader_id} online with funds...")

    # 2. Trading Loop
    while True:
        mid = market.update()
        
        # Decide Side: BUY at/below mid, SELL at/above mid
        side = random.choice(["BUY", "SELL"])
        
        if side == "BUY":
            price = mid - random.randint(0, PRICE_SPREAD)
            color = "\033[92m" # Green
        else:
            price = mid + random.randint(0, PRICE_SPREAD)
            color = "\033[91m" # Red

        quantity = random.randint(1, 5) # 1-5 BTC
        
        # Scale for the API
        scaled_price = price * DECIMALS
        scaled_quantity = quantity * DECIMALS
        
        try:
            start = time.time()
            res = session.post(
                f"{BASE_URL}/order", 
                json={"side": side, "quantity": scaled_quantity, "price": scaled_price},
                headers=headers,
                timeout=5
            )
            latency = (time.time() - start) * 1000

            if res.status_code in [200, 201]:
                print(f"{color}[{side}]{trader_id:2d} | {quantity:2d} BTC @ ${price:5d} | {latency:4.1f}ms\033[0m")
            else:
                print(f"\033[93m[WARN] {res.status_code} {res.text}\033[0m")

        except Exception as e:
            print(f"\033[31m[CRITICAL] {e}\033[0m")

        # Control trading frequency
        time.sleep(1.0 / ORDERS_PER_SEC)

if __name__ == "__main__":
    print(f"--- Launching Load Test: {NUM_TRADERS} Traders ---")
    
    # We use a ThreadPool to manage all traders in one Python process
    with ThreadPoolExecutor(max_workers=NUM_TRADERS) as executor:
        for i in range(NUM_TRADERS):
            executor.submit(trader_logic, i)
            # Staggered start to prevent thundering herd on login
            time.sleep(0.1)