import requests
import json

url = "http://localhost:2379/kv"

for i in range(1, 1200):
    keyString = "key" + str(i)
    valString = "value" + str(i)
    payload = json.dumps({
    "key": keyString,
    "value": valString
    })
    headers = {
    'Content-Type': 'application/json'
    }

    print(url)
    response = requests.request("PUT", url , headers=headers, data=payload)

    print(response.text)
