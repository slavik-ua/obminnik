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

def send_orders(token):
    headers = {"Authorization": f"Bearer {token}"}
    print(f"--- Starting Load Test on {BASE_URL} ---")
    
    while True:
        side = random.choice(["BUY", "SELL"])
        price = random.randint(95, 105) 
        quantity = random.randint(1, 10)

        order_data = {
            "side": side,
            "quantity": int(quantity),
            "price": int(price)
        }

        try:
            start_time = time.time()
            response = requests.post(f"{BASE_URL}/order", json=order_data, headers=headers)
            
            if response.status_code == 200 or response.status_code == 201:
                print(f"[SUCCESS] {side} {quantity} @ {price} | Status: {response.status_code}")
            else:
                print(f"[ERROR] Status {response.status_code}: {response.text}")

        except Exception as e:
            print(f"[CRITICAL] Connection failed: {e}")

        time.sleep(random.uniform(0.1, 0.5))

if __name__ == "__main__":
    jwt_token = setup_auth()
    send_orders(jwt_token)