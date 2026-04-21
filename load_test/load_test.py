import requests
import time
import random

BASE_URL = "http://127.0.0.1:8000"
USER_EMAIL = f"trader_{random.randint(1, 1000)}@example.com"
PASSWORD = "password123"

def setup_auth():
    print(f"--- Registering {USER_EMAIL} ---")
    reg_res = requests.post(f"{BASE_URL}/register", json={
        "email": USER_EMAIL,
        "password": PASSWORD
    })
    
    print("--- Logging in ---")
    login_res = requests.post(f"{BASE_URL}/login", json={
        "email": USER_EMAIL,
        "password": PASSWORD
    })
    
    if login_res.status_code != 200:
        print("Login failed!", login_res.text)
        exit(1)
        
    return login_res.json().get("token")

class PriceSimulator:
    def __init__(self, start_price=200.0, volatility=0.6, spread=2.0):
        self.current_price = start_price
        self.volatility = volatility
        self.spread = spread
        self.trend = 0.0
        
    def get_next_prices(self):
        # Apply random walk with slight trend momentum
        self.trend = (self.trend * 0.9) + random.uniform(-0.1, 0.1)
        change = random.uniform(-self.volatility, self.volatility) + self.trend
        self.current_price += change
        
        # Keep price within reasonable bounds
        self.current_price = max(10.0, min(1000.0, self.current_price))
        
        # Calculate bid/ask around current price
        mid = round(self.current_price, 2)
        bid = round(mid - self.spread/2)
        ask = round(mid + self.spread/2)
        
        # Ensure gap exists
        if bid >= ask:
            ask = bid + 1
            
        return bid, ask

def send_orders(token):
    headers = {"Authorization": f"Bearer {token}"}
    print(f"--- Starting Advanced Load Test on {BASE_URL} ---")
    
    sim = PriceSimulator(start_price=200.0)
    
    while True:
        bid, ask = sim.get_next_prices()
        
        # Randomly choose to send a BUY or SELL order
        side = random.choice(["BUY", "SELL"])
        price = bid if side == "BUY" else ask
        quantity = random.randint(1, 20)

        order_data = {
            "side": side,
            "quantity": int(quantity),
            "price": int(price)
        }

        try:
            response = requests.post(f"{BASE_URL}/order", json=order_data, headers=headers)
            
            if response.status_code == 200 or response.status_code == 201:
                col = "\033[92m" if side == "BUY" else "\033[91m"
                reset = "\033[0m"
                print(f"{col}[{side}]{reset} {quantity:2d} @ {price:4d} | Market: {sim.current_price:.2f}")
            else:
                print(f"[ERROR] Status {response.status_code}: {response.text}")

        except Exception as e:
            print(f"[CRITICAL] Connection failed: {e}")

        # High frequency updates
        time.sleep(random.uniform(0.05, 0.2))

if __name__ == "__main__":
    jwt_token = setup_auth()
    send_orders(jwt_token)