const TRACKING_SERVER = 'http://localhost:8080'; // 実際のトラッキングサーバーのURLに変更してください

let userId = null;

function getUserId() {
    if (userId) return Promise.resolve(userId);

    const storedUserId = localStorage.getItem('trackingUserId');
    if (storedUserId) {
        userId = storedUserId;
        return Promise.resolve(userId);
    }

    return fetch(`${TRACKING_SERVER}/generate-user-id`)
        .then(response => response.json())
        .then(data => {
            userId = data.userId;
            localStorage.setItem('trackingUserId', userId);
            return userId;
        })
        .catch(error => console.error('Error generating user ID:', error));
}

function track(event) {
    getUserId().then(userId => {
        fetch(`${TRACKING_SERVER}/track`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                userId: userId,
                event: event,
                timestamp: new Date().toISOString(),
            }),
        });
    });
}

document.addEventListener('DOMContentLoaded', () => {
    track('pageview');
});
