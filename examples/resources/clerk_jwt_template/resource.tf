resource "clerk_jwt_template" "example" {
  name   = "my-custom-jwt"
  claims = jsonencode({
    email  = "{{user.primary_email_address}}"
    org_id = "{{org.id}}"
  })
  lifetime           = 60
  allowed_clock_skew = 5
  signing_algorithm  = "RS256"
}
