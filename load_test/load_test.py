import requests
import time
import random
import threading
from concurrent.futures import ThreadPoolExecutor

# --- Configuration ---
BASE_URL = "http://127.0.0.1:8000"
NUM_TRADERS = 20
ORDERS_PER_SEC = 5    # Speed per trader
PRICE_SPREAD = 2      # How tight the market is (smaller = more matches)

# --- Shared Market State ---
# This ensures all traders fight over the same price levels
class GlobalMarket:
    def __init__(self):
        self.mid_price = 200.0
        self.lock = threading.Lock()

    def update(self):
        with self.lock:
            # Random walk: move price by -1, 0, or 1
            self.mid_price += random.uniform(-1, 1)
            self.mid_price = max(10, min(1000, self.mid_price))
            return int(self.mid_price)

market = GlobalMarket()

def trader_logic(trader_id):
    """ Main loop for a single virtual trader """
    email = f"bot_{trader_id}_{random.randint(1000, 9999)}@exchange.com"
    password = "password123"
    session = requests.Session() # Uses Keep-Alive for high performance

    # 1. Setup Auth
    try:
        session.post(f"{BASE_URL}/register", json={"email": email, "password": password})
        login_res = session.post(f"{BASE_URL}/login", json={"email": email, "password": password})
        token = login_res.json().get("token")
        headers = {"Authorization": f"Bearer {token}"}
    except Exception as e:
        print(f"Trader {trader_id} failed auth: {e}")
        return

    print(f"Trader {trader_id} online and trading...")

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

        quantity = random.randint(1, 10)
        
        try:
            start = time.time()
            res = session.post(
                f"{BASE_URL}/order", 
                json={"side": side, "quantity": quantity, "price": price},
                headers=headers,
                timeout=5
            )
            latency = (time.time() - start) * 1000

            if res.status_code in [200, 201]:
                print(f"{color}[{side}]{trader_id:2d} | {quantity:2d} @ {price:4d} | {latency:4.1f}ms\033[0m")
            else:
                print(f"\033[93m[WARN] {res.status_code}\033[0m")

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