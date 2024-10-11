#!/bin/bash
FQDN="HADOOP.COM"
ADMIN="admin"
PASS="admin123@"
KERBEROS_USER="app"
KRB5_KTNAME=/etc/admin.keytab
cat /etc/hosts
echo "hostname: ${FQDN}"
inited="/app/inited"
function init_user() {
	if [ -f "${inited}" ];then
		echo "user inited"
	        kadmin.local -q "xst -k /app/cli.keytab -norandkey cli"
	        kadmin.local -q "xst -k /app/${KERBEROS_USER}.keytab -norandkey ${KERBEROS_USER}"
		return;
	fi
	echo "begin init user"
	# create kerberos database
	echo -e "${PASS}\n${PASS}" | kdb5_util create -s
	# create admin
	echo -e "${PASS}\n${PASS}" | kadmin.local -q "addprinc ${ADMIN}/admin"
	# create hadoop
	echo -e "${PASS}\n${PASS}" | kadmin.local -q "addprinc cli"
	echo -e "${PASS}\n${PASS}" | kadmin.local -q "addprinc ${KERBEROS_USER}"
	kadmin.local -q "ktadd -norandkey -k ${KRB5_KTNAME} cli"
	kadmin.local -q "ktadd -norandkey -k ${KRB5_KTNAME} ${KERBEROS_USER}"
	kadmin.local -q "xst -k /app/cli.keytab -norandkey cli"
	kadmin.local -q "xst -k /app/${KERBEROS_USER}.keytab -norandkey ${KERBEROS_USER}"
	touch "${inited}"
	echo "user inite success"
}
function main() {
	init_user
	/usr/bin/supervisord -n -c /etc/supervisord.conf
}
main
