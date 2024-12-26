package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"golang.org/x/sys/unix"
)

func PidDaemon(setpoint float64) {
	// Kqueue 事件驱动读取温度
	kq, err := unix.Kqueue()
	if err != nil {
		fmt.Printf("创建 Kqueue 失败: %v\n", err)
		return
	}
	defer unix.Close(kq)

	// 设置监听温度文件
	tempFile := "/path/to/temperature/file"
	tempFd, err := os.Open(tempFile)
	if err != nil {
		fmt.Printf("打开温度文件失败: %v\n", err)
		return
	}
	defer tempFd.Close()

	// 创建 Kqueue 事件
	kev := unix.Kevent_t{
		Ident:  uint64(tempFd.Fd()),
		Filter: unix.EVFILT_VNODE, // 监听文件变化事件
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
		Fflags: unix.NOTE_WRITE, // 监听写入事件
	}

	// 开始监听温度文件
	for {
		// 等待事件触发
		_, err := unix.Kevent(kq, []unix.Kevent_t{kev}, nil, nil)
		if err != nil {
			fmt.Printf("Kqueue 事件失败: %v\n", err)
			continue
		}

		// 读取温度数据
		temperature, timestamp, err := readTemperature(tempFile)
		if err != nil {
			fmt.Printf("读取温度失败: %v\n", err)
			continue
		}

		// 检查是否有 PID 参数更新
		if globalPID.updated {
			fmt.Println("PID 参数已更新，重新加载...")
			globalPID.ResetUpdateFlag()
		}

		// 使用 PID 计算加热时间和风扇状态
		heatingOutput := globalPID.Calculate(setpoint, temperature, timestamp)

		// 控制加热
		if heatingOutput > 0 {
			go controlHeating(heatingOutput)
		}

		// 风扇控制：温度过高时开启风扇，降到设定温度以下时关闭风扇
		if temperature > setpoint {
			go controlCooling(true) // 启动风扇降温
		} else {
			go controlCooling(false) // 关闭风扇
		}

		// 输出调试信息
		fmt.Printf("当前温度: %.2f°C, PID 加热输出: %.2f, 时间: %d\n", temperature, heatingOutput, timestamp)
	}
}

// 启动 Web 服务
func StartWebServer() {
	http.HandleFunc("/pid", httpHandler)
	fmt.Println("Starting WebServer on port 8080...")
	http.ListenAndServe(":8080", nil)
}

// 其他函数保持不变

type PID struct {
	Kp, Ki, Kd    float64
	previousError float64
	integral      float64
	lock          sync.Mutex
	updated       bool // 参数更新标志
}

func (pid *PID) Calculate(setpoint, temperature float64, timestamp int64) float64 {
	pid.lock.Lock()
	defer pid.lock.Unlock()

	error := setpoint - temperature
	pid.integral += error
	derivative := error - pid.previousError
	output := pid.Kp*error + pid.Ki*pid.integral + pid.Kd*derivative
	pid.previousError = error

	return output
}

var globalPID = PID{
	Kp: 2.0,
	Ki: 0.5,
	Kd: 0.1,
}

func (pid *PID) UpdateParams(kp, ki, kd float64) {
	pid.lock.Lock()
	defer pid.lock.Unlock()
	pid.Kp, pid.Ki, pid.Kd = kp, ki, kd
	pid.updated = true
}

func (pid *PID) GetParams() (kp, ki, kd float64) {
	pid.lock.Lock()
	defer pid.lock.Unlock()
	return pid.Kp, pid.Ki, pid.Kd
}

func (pid *PID) ResetUpdateFlag() {
	pid.lock.Lock()
	defer pid.lock.Unlock()
	pid.updated = false
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var params struct {
			Kp float64 `json:"kp"`
			Ki float64 `json:"ki"`
			Kd float64 `json:"kd"`
		}
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		globalPID.UpdateParams(params.Kp, params.Ki, params.Kd)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("PID parameters updated successfully"))
	} else if r.Method == http.MethodGet {
		kp, ki, kd := globalPID.GetParams()
		resp := fmt.Sprintf(`{"kp":%.2f, "ki":%.2f, "kd":%.2f}`, kp, ki, kd)
		w.Write([]byte(resp))
	} else {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
	}
}

// 控制加热
func controlHeating(output float64) {
	// 实现加热控制逻辑
	fmt.Printf("控制加热: 输出 %.2f\n", output)
}

// 控制风扇
func controlCooling(state bool) {
	if state {
		fmt.Println("启动风扇降温")
		// 实现启动风扇逻辑
	} else {
		fmt.Println("关闭风扇")
		// 实现关闭风扇逻辑
	}
}

// 读取温度数据，解析文件内容
func readTemperature(path string) (float64, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("无法打开温度文件: %v", err)
	}
	defer file.Close()

	var temperature float64
	var timestamp int64
	_, err = fmt.Fscanf(file, "%f,%d\n", &temperature, &timestamp)
	if err != nil {
		return 0, 0, fmt.Errorf("读取温度数据失败: %v", err)
	}

	return temperature, timestamp, nil
}
