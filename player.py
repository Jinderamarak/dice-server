import asyncio

import jwt
import requests
import websockets

import play

API_URL="localhost:9000/api"

def create_new_farkle_game() -> str:
    """Creates a new game and returns the token."""
    try:
        response = requests.post(f"http://{API_URL}/farkle")
        response.raise_for_status()  # Raise HTTPError for bad responses (4xx or 5xx)
        data = response.json()
        token = data.get("token")
        if not token:
            print("Token not found in response.")
            raise SystemExit
        return token
    except requests.exceptions.RequestException as e:
        print(f"HTTP request failed: {e}")
        raise SystemExit

def join_existing_farkle_game(gameId: str) -> str:
    """Joins an existing game and returns the token."""
    try:
        response = requests.post(f"http://{API_URL}/farkle/{gameId}/join")
        response.raise_for_status()
        data = response.json()
        token = data.get("token")
        if not token:
            print("Token not found in response.")
            raise SystemExit
        return token
    except requests.exceptions.RequestException as e:
        print(f"HTTP request failed: {e}")
        raise SystemExit

def extract_jwt_payload(token: str) -> dict:
    """Extracts and returns the payload from a JWT token."""
    try:
        payload = jwt.decode(token, options={"verify_signature": False})
        return payload
    except Exception as e:
        print(f"Error decoding JWT: {e}")
        raise SystemExit

async def websocket_listener(websocket_url: str, user_id: str):
    """Listens to a WebSocket and prints received messages."""
    try:
        async with websockets.connect(websocket_url) as websocket:
            # print(f"Connected to WebSocket: {websocket_url}")

            state = dict()
            state["playerId"] = user_id
            await websocket.send(play.start_game(state))

            async for message in websocket:
                try:
                    outbound = play.handle_message(message, state)
                    for out in outbound:
                        await websocket.send(out)
                except SystemExit:
                    return
    except Exception as e:
        print(f"WebSocket error: {e}")
        # raise SystemExit

async def play_game(id: int):
    print(f"Starting game > {id}")

    p1_token = create_new_farkle_game()

    payload = extract_jwt_payload(p1_token)
    p1_user_id = payload["userId"]
    game_id = payload["gameId"]

    p2_token = join_existing_farkle_game(game_id)
    payload = extract_jwt_payload(p2_token)
    p2_user_id = payload["userId"]

    server_host = payload["serverHost"]
    p1_url = f"ws://{server_host}/api/game/farkle/{p1_token}"
    p2_url = f"ws://{server_host}/api/game/farkle/{p2_token}"
    await asyncio.gather(
        websocket_listener(p1_url, p1_user_id),
        websocket_listener(p2_url, p2_user_id)
    )

CONCURRENT_LIMIT = 1000
TOTAL_COUNT = 5000

async def run_games_concurrently(num_games):
    semaphore = asyncio.Semaphore(CONCURRENT_LIMIT)

    async def run_single_game(id: int):
        async with semaphore:
            await play_game(id)

    tasks = [run_single_game(i) for i in range(num_games)]
    await asyncio.gather(*tasks)

if __name__ == "__main__":
    asyncio.run(run_games_concurrently(TOTAL_COUNT))