#ifndef COMMON_H
#define COMMON_H

#include <QString>
#include <QtWebSockets/QWebSocket>

const QString WEBSOCKET_SERVER_HOST = "WEBSOCKET_SERVER_HOST";
const QString WEBSOCKET_SERVER_PORT = "WEBSOCKET_SERVER_PORT";
const QString WEBSOCKET_USER_NAME = "WEBSOCKET_USER_NAME";
const QString WEBSOCKET_USER_PWD = "WEBSOCKET_USER_PWD";
const QString WEBSOCKET_REMBER_PWD = "WEBSOCKET_REMBER_PWD";

typedef struct _user_info {
    QString strUserName;
    QString strUserId;
    QString strPwd;
    QString strLoginTime;
    QString strEmail;
} UserInfo, *PUserInfo;

typedef struct _msg_info {
    QString strEmail;
    QString strUserName;
    QString strUserId;
    QString strMsg;
    QString strTime;
    QString fileLink;
} MsgInfo, *PMsgInfo;

enum HttpRequest {
    REQUEST_LOGIN,
    REQUEST_REGISTER,
    REQUEST_UPLOAD_FILE,
    REQUEST_DOWNLOAD_FILE
};

const QString WEBSOCKET_ERROR_STRINGS[24] = {
    "An unidentified error occurred.",
    "The connection was refused by the peer (or timed out).",
    "The remote host closed the connection. Note that the client socket (i.e., this socket) will be closed after the remote close notification has been sent.",
    "The host address was not found.",
    "The socket operation failed because the application lacked the required privileges.",
    "The local system ran out of resources (e.g., too many sockets).",
    "The socket operation timed out.",
    "The datagram was larger than the operating system's limit (which can be as low as 8192 bytes).",
    "An error occurred with the network (e.g., the network cable was accidentally plugged out).",
    "The address specified to QAbstractSocket::bind() is already in use and was set to be exclusive.",
    "The address specified to QAbstractSocket::bind() does not belong to the host.",
    "The requested socket operation is not supported by the local operating system (e.g., lack of IPv6 support).",
    "The socket is using a proxy, and the proxy requires authentication.",
    "The SSL/TLS handshake failed, so the connection was closed (only used in QSslSocket)",
    "Used by QAbstractSocketEngine only, The last operation attempted has not finished yet (still in progress in the background).",
    "Could not contact the proxy server because the connection to that server was denied",
    "The connection to the proxy server was closed unexpectedly (before the connection to the final peer was established)",
    "The connection to the proxy server timed out or the proxy server stopped responding in the authentication phase.",
    "The proxy address set with setProxy() (or the application proxy) was not found.",
    "The connection negotiation with the proxy server failed, because the response from the proxy server could not be understood.",
    "An operation was attempted while the socket was in a state that did not permit it.",
    "The SSL library being used reported an internal error. This is probably the result of a bad installation or misconfiguration of the library.",
    "Invalid data (certificate, key, cypher, etc.) was provided and its use resulted in an error in the SSL library.",
    "A temporary error occurred (e.g., operation would block and socket is non-blocking)."};

extern UserInfo g_stUserInfo;

extern QWebSocket g_WebSocket;


#endif // COMMON_H
