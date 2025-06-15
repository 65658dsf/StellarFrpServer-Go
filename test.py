from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/api/v1/nodes/load/webhook', methods=['POST'])
def NodeLoadWebHook():
    response = request.get_json()
    print(response)
    return jsonify(code=200, message="success")

if __name__ == '__main__':
    app.run(debug=True)