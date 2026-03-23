import UserNotifications

func sendNotification(title: String, body: String) {
    let center = UNUserNotificationCenter.current()
    center.requestAuthorization(options: [.alert]) { _, _ in }

    let content = UNMutableNotificationContent()
    content.title = title
    content.body = body

    let request = UNNotificationRequest(identifier: UUID().uuidString, content: content, trigger: nil)
    center.add(request)
}
