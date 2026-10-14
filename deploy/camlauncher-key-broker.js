const TAILSCALE_OAUTH_URL = 'https://api.tailscale.com/api/v2/oauth/token'
const TAILSCALE_KEY_URL = 'https://api.tailscale.com/api/v2/tailnet/-/keys'

const json = (body, status = 200) => new Response(JSON.stringify(body), {
  status,
  headers: {
    'content-type': 'application/json; charset=utf-8',
    'cache-control': 'no-store',
  },
})

export default {
  async fetch(request, env) {
    const path = new URL(request.url).pathname
    if (request.method === 'GET' && path === '/health') {
      return json({ service: 'camlauncher-key-broker', ready: Boolean(env.TAILSCALE_CLIENT_ID && env.TAILSCALE_CLIENT_SECRET) })
    }
    if (request.method !== 'POST' || path !== '/v1/tunnel-key') {
      return json({ error: 'not_found' }, 404)
    }

    const clientIP = request.headers.get('CF-Connecting-IP') || 'unknown'
    try {
      const tokenResponse = await fetch(TAILSCALE_OAUTH_URL, {
        method: 'POST',
        headers: { 'content-type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
          grant_type: 'client_credentials',
          client_id: env.TAILSCALE_CLIENT_ID,
          client_secret: env.TAILSCALE_CLIENT_SECRET,
        }),
      })
      if (!tokenResponse.ok) throw new Error(`oauth_${tokenResponse.status}`)
      const { access_token } = await tokenResponse.json()
      if (!access_token) throw new Error('missing_access_token')

      const keyResponse = await fetch(TAILSCALE_KEY_URL, {
        method: 'POST',
        headers: {
          authorization: `Bearer ${access_token}`,
          'content-type': 'application/json',
        },
        body: JSON.stringify({
          capabilities: {
            devices: {
              create: {
                reusable: false,
                ephemeral: true,
                preauthorized: true,
                tags: ['tag:camlauncher'],
              },
            },
          },
          expirySeconds: 300,
          description: `CamLauncher alpha tunnel node (${clientIP})`,
        }),
      })
      if (!keyResponse.ok) throw new Error(`key_${keyResponse.status}`)
      const { key } = await keyResponse.json()
      if (!key) throw new Error('missing_auth_key')
      console.log(JSON.stringify({ event: 'tunnel_key_issued', clientIP }))
      return json({ authKey: key, target: '100.117.75.101:25565' })
    } catch (error) {
      console.error(JSON.stringify({ event: 'tunnel_key_failed', clientIP, error: String(error) }))
      return json({ error: 'tunnel_unavailable', detail: String(error).slice(0, 160) }, 503)
    }
  },
}
