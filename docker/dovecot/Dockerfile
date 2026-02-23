FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    dovecot-core \
    dovecot-imapd \
    dovecot-managesieved \
    dovecot-sieve \
    && rm -rf /var/lib/apt/lists/*

# Create vmail user for virtual mailboxes
RUN groupadd -g 5000 vmail && useradd -u 5000 -g vmail -d /var/mail/vhosts -s /sbin/nologin vmail

# Create mail directories
RUN mkdir -p /var/mail/vhosts/testuser/Maildir/{cur,new,tmp} && \
    chown -R vmail:vmail /var/mail/vhosts

# Copy configuration files
COPY dovecot.conf /etc/dovecot/dovecot.conf
COPY 10-auth.conf /etc/dovecot/conf.d/10-auth.conf
COPY 10-mail.conf /etc/dovecot/conf.d/10-mail.conf
COPY 10-master.conf /etc/dovecot/conf.d/10-master.conf
COPY 20-imap.conf /etc/dovecot/conf.d/20-imap.conf
COPY 90-sieve.conf /etc/dovecot/conf.d/90-sieve.conf

# Create passwd-file for virtual users
# Format: user:{PLAIN}password:uid:gid::home::
RUN echo "testuser:{PLAIN}testpass:5000:5000::/var/mail/vhosts/testuser::" > /etc/dovecot/users

# Create sieve directory for testuser
RUN mkdir -p /var/mail/vhosts/testuser/sieve && \
    chown -R vmail:vmail /var/mail/vhosts/testuser

EXPOSE 143 4190

CMD ["/usr/sbin/dovecot", "-F"]
