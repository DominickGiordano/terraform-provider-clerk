# Add a satellite domain for a production application
resource "clerk_domain" "app" {
  name         = "app.example.com"
  is_satellite = true
}

# Add a satellite domain with a proxy
resource "clerk_domain" "proxied" {
  name         = "secure.example.com"
  is_satellite = true
  proxy_url    = "https://proxy.example.com/__clerk"
}
