import json
from typing import List, Any, Dict

def start_game(state: Dict[str, Any]) -> str:
    player_id = ""
    if "playerId" in state:
        player_id = state["playerId"]

    return json.dumps({
        "variant": "farkle-player-ready",
        "data": {
            "playerId": player_id,
        }
    })

def handle_message(message: str, state: Dict[str, Any]) -> List[str]:
    data = json.loads(message)
    variant = data["variant"]
    # print(f"-> {variant}")

    is_me = False
    if "playerId" in data["data"]:
        is_me = data["data"]["playerId"] == state["playerId"]

    if variant == "farkle-game-begin":
        # print("Game started!")
        pass
    elif variant == "farkle-sync-state":
        pass
    elif variant == "farkle-turn-begin":
        state["isMyTurn"] = is_me
        # if is_me:
        #     print("My turn!")
        # else:
        #     print("Enemy's turn!")
    elif variant == "farkle-dice-roll":
        if state["isMyTurn"]:
            if data["data"]["busted"]:
                # print("Oh no! Busted!")
                return []

            turn = list()
            for dice in data["data"]["dice"]:
                if dice["value"] == 1 or dice["value"] == 5:
                    turn.append(json.dumps({"variant": "farkle-dice-touch", "data": {"diceId": dice["id"], "selected": True}}))
            turn.append(json.dumps({"variant": "farkle-end-turn", "data": {"playerId": state["playerId"]}}))
            return turn
    elif variant == "farkle-dice-touch":
        pass
    elif variant == "farkle-dice-touched":
        pass
    elif variant == "farkle-update-score":
        my_score = data["data"]["playerId"] == state["playerId"]
        total = data["data"]["scores"]["total"]
        turn = data["data"]["scores"]["turn"]
        selected = data["data"]["scores"]["selected"]
        player = "My" if my_score else "Enemy"
        # print(f"{player} score: {total} (turn: {turn}, selected: {selected})")
    elif variant == "farkle-turn-timeout":
        # if is_me:
        #     print("I ran out of time!")
        # else:
        #     print("Enemy ran out of time!")
        pass
    elif variant == "farkle-score-roll":
        # if is_me:
        #     print("I am rolling again!")
        # else:
        #     print("Enemy is rolling again!")
        pass
    elif variant == "farkle-end-turn":
        # if is_me:
        #     print("I ended my turn!")
        # else:
        #     print("Enemy ended their turn!")
        pass
    elif variant == "farkle-game-end":
        is_winner = data["data"]["winnerId"] == state["playerId"]
        # if is_winner:
        #     print("I won!")
        # else:
        #     print("I lost!")
    elif variant == "farkle-error":
        pass
    elif variant == "farkle-terminate":
        raise SystemExit

    return []