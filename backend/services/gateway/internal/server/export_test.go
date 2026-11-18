package server

// PlainTextForTest expose le nettoyage des extraits aux tests du paquet server_test.
func PlainTextForTest(s string) string { return plainText(s) }
