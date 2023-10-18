data "kerberos_token" "example" {
  username = "Administrator"
  password = "Test1234!"
  realm    = "TEST.LAN"
  service  = "HTTP/test.lan"
  kdc      = "localhost:1088"
}

output "token" {
  sensitive = true
  value     = kerberos_token.example.token
}
