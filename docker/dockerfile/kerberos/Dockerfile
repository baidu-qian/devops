FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive
ENV TIME_ZONE=Asia/Shanghai
#RUN sed -i s@/archive.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list
#RUN sed -i s@/security.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list
#RUN apt clean
#RUN apt update
#RUN apt install -y ntp python3 python3-pip python3-setuptools python3-wheel python3-dev    krb5-admin-server krb5-kdc
#RUN apt install vim -y
#RUN rm -rf /var/lib/apt/lists/*
#RUN mkdir -p /var/log/supervisord/
#RUN mkdir /app/
##RUN pip3 install supervisor==4.2.4
#RUN pip3 install supervisor

RUN sed -i s@/archive.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list \ 
    && sed -i s@/security.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list \
    && apt clean \
    && apt update \
    && apt install -y ntp python3 python3-pip python3-setuptools python3-wheel python3-dev    krb5-admin-server krb5-kdc vim supervisor iputils-ping net-tools tzdata\
    && rm -rf /var/lib/apt/lists/*  \
    && mkdir -p /var/log/supervisord/  \
    && mkdir -p /app/  \
    && ln -snf /usr/share/zoneinfo/$TIME_ZONE /etc/localtime && echo $TIME_ZONE > /etc/timezone \
    && dpkg-reconfigure -f noninteractive tzdata \
    && apt-get clean \
    && rm -rf /tmp/* /var/cache/* /usr/share/doc/* /usr/share/man/* /var/lib/apt/lists/*
          
COPY ./server/krb5.conf /etc/krb5kdc/kdc.conf
COPY ./server/kadm5.acl /etc/krb5kdc/kadm5.acl
COPY ./client/krb5.conf /etc/krb5.conf
COPY ./start.sh /start.sh
COPY ./supervisord.conf /etc/supervisord.conf
WORKDIR /app
CMD ["/bin/bash", "/start.sh"]
