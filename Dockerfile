#
#   Author: Rohith (gambol99@gmail.com)
#   Date: 2014-12-11 13:33:17 +0000 (Thu, 11 Dec 2014)
#
#  vim:ts=2:sw=2:et
#
FROM busybox
MAINTAINER <gambol99@gmail.com>

ADD ./stage/config-store /bin/config-store
ADD ./stage/startup.sh ./startup.sh

RUN chmod +x ./startup.sh; chmod +x /bin/config-store

ENTRYPOINT [ "./startup.sh" ]

