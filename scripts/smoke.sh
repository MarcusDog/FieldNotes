#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export FIELDNOTES_TEST_ROOT="$ROOT"
python3 - <<'PY'
import json, os, pathlib, socket, subprocess, tempfile, time, urllib.request, urllib.error, uuid
root = pathlib.Path(os.environ['FIELDNOTES_TEST_ROOT'])
with tempfile.TemporaryDirectory(prefix='fieldnotes-smoke-') as temp:
    temp = pathlib.Path(temp)
    binary = temp / 'fieldnotes'
    subprocess.run(['go', 'build', '-o', str(binary), './cmd/fieldnotes'], cwd=root/'server', check=True)
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0)); port = sock.getsockname()[1]
    url = f'http://127.0.0.1:{port}'
    env = dict(os.environ, ADDR=f'127.0.0.1:{port}', DATA_FILE=str(temp/'issues.json'))
    process = None
    def request(path, method='GET', payload=None, headers=None):
        body = None if payload is None else json.dumps(payload).encode()
        req = urllib.request.Request(url+path, data=body, method=method, headers={'Content-Type':'application/json', **(headers or {})})
        try:
            with urllib.request.urlopen(req, timeout=4) as response:
                return response.status, json.load(response), response.headers
        except urllib.error.HTTPError as error:
            return error.code, json.load(error), error.headers
    def start():
        global process
        process = subprocess.Popen([str(binary)], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        for _ in range(50):
            if process.poll() is not None: raise RuntimeError('Server exited before health check')
            try:
                if request('/api/health')[0] == 200: return
            except (urllib.error.URLError, TimeoutError): pass
            time.sleep(.1)
        raise RuntimeError('Server did not become healthy')
    def stop():
        if process is not None and process.poll() is None:
            process.terminate(); process.wait(timeout=10)
    try:
        start()
        assert request('/api/issues')[1] == {'data': []}
        payload = {'title':'[API smoke] 入口墙面破损', 'description':'这是显式 API 测试，不是真机上报', 'priority':'high', 'position':{'x':.42,'y':.03,'z':1.26}}
        headers = {'Idempotency-Key':str(uuid.uuid4())}
        status, created, _ = request('/api/issues', 'POST', payload, headers)
        assert status == 201, created
        issue = created['data']; assert issue['status'] == 'open' and issue['id']
        assert request('/api/issues', 'POST', payload, headers)[1]['data']['id'] == issue['id']
        assert len(request('/api/issues')[1]['data']) == 1
        path = '/api/issues/'+issue['id']+'/status'
        status, updated, _ = request(path, 'PATCH', {'status':'in_progress'}, {'Origin':url})
        assert status == 200 and updated['data']['status'] == 'in_progress', updated
        assert request(path, 'PATCH', {'status':'unknown'})[0] == 400
        assert request('/api/issues', 'POST', dict(payload, title='  '))[0] == 400
        assert request('/api/issues', 'POST', dict(payload, priority='urgent'))[0] == 400
        assert request('/api/issues', headers={'Origin':'http://localhost:5173'})[2]['Access-Control-Allow-Origin'] == 'http://localhost:5173'
        stop(); start()
        persisted = request('/api/issues')[1]['data']
        assert len(persisted) == 1 and persisted[0]['id'] == issue['id'] and persisted[0]['status'] == 'in_progress'
        assert request('/api/issues', 'POST', payload, headers)[1]['data']['id'] == issue['id']
        assert len(request('/api/issues')[1]['data']) == 1
        print('PASS: health → create → list → same-origin status update → actual process restart → persisted issue/status → deduplicated retry')
        print('PASS: blank-title/enum errors and development CORS; disposable test data isolated from real reports')
    finally: stop()
PY
