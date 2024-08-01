#!/bin/bash -eu

export PATH=/usr/bin:/bin:/usr/sbin:/sbin

tuser=tsvc
name=telemetry-admin
_localstatedir=/var
_tsvcdir=${_localstatedir}/lib/${tuser}
_certdir=${_tsvcdir}/certs

[ -d ${_certdir} ] && cp -a ${_certdir}/* /etc/pki/trust/anchors

update-ca-certificates

if [[ ! -d ${_tsvcdir} ]]; then
    echo "Error: no ${_tsvcdir} directory found; did you bind mount the service account dir to ${_tsvcdir}?"
    exit 1
fi

uid_gid=(
    $(stat -c '%u %g' ${_tsvcdir})
)

# ensure that the required group exists or add it with matching gid
# if needed
getent group ${tuser} >/dev/null || groupadd \
    -r \
    -g ${uid_gid[1]} \
    ${tuser}

# ensure that the required user exists or add it with matching uid
# if needed
getent passwd ${tuser} >/dev/null || useradd \
    -r \
    -g ${tuser} \
    -u ${uid_gid[0]} \
    -d ${_tsvcdir} \
    -s /sbin/nologin \
    -c "user for ${name}" ${tuser}

chown -R ${uid_gid[0]}:${uid_gid[1]} ${_tsvcdir}

cmd_args=(
    /usr/bin/${name}
    "${@}"
)

su - ${tuser} --shell /bin/bash -c "${cmd_args[*]}"
