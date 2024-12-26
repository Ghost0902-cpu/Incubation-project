 #include <stdio.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/event.h>
#include <time.h>
#include <string.h>

#define TEMP_FILE "/tmp/temperature_data"

void write_temperature(double temperature) {
    // 模拟温度和时间戳写入
    int fd = open(TEMP_FILE, O_WRONLY | O_CREAT | O_TRUNC, 0644);
    if (fd < 0) {
        perror("Failed to open temperature file");
        return;
    }

    char buffer[256];
    uint64_t timestamp = (uint64_t)time(NULL);
    snprintf(buffer, sizeof(buffer), "%.2f,%lu\n", temperature, timestamp);
    write(fd, buffer, strlen(buffer));
    close(fd);
}

int main() {
    int kq = kqueue();
    if (kq == -1) {
        perror("Failed to create kqueue");
        return 1;
    }

    struct kevent event;
    int temp_fd = open(TEMP_FILE, O_RDONLY | O_CREAT, 0644);
    if (temp_fd == -1) {
        perror("Failed to open temperature file");
        return 1;
    }

    EV_SET(&event, temp_fd, EVFILT_VNODE, EV_ADD | EV_CLEAR, NOTE_WRITE, 0, NULL);
    if (kevent(kq, &event, 1, NULL, 0, NULL) == -1) {
        perror("Failed to register event");
        return 1;
    }

    printf("Waiting for temperature updates...\n");

    while (1) {
        struct kevent triggered_event;
        int n = kevent(kq, NULL, 0, &triggered_event, 1, NULL);
        if (n > 0) {
            double temperature = 36.5 + ((rand() % 100) / 100.0); // 模拟随机温度
            write_temperature(temperature);
            printf("Updated temperature: %.2f\n", temperature);
        }
    }

    close(temp_fd);
    return 0;
}
