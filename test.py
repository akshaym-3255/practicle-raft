import requests
import json

url = "http://localhost:2379/kv"

for i in range(10):
    keyString = "key" + str(i)
    valString = "value" + str(i)
    payload = json.dumps({
    keyString: valString
    })
    headers = {
    'Content-Type': 'application/json'
    }

    print(url)
    response = requests.request("PUT", url + "/" + keyString, headers=headers, data=payload)

    print(response.text)
