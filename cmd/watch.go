package cmd

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/no8ge/sidecar/pkg/k8s"
	"github.com/no8ge/sidecar/pkg/minio"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.PersistentFlags().StringP("dir", "d", "./tmp", "the dir for sidecar watch")
	watchCmd.PersistentFlags().StringP("endpoint", "e", "172.16.60.10:31478", "endpoint of minio")
	watchCmd.PersistentFlags().StringP("accessKeyID", "a", "admin", "accessKeyID of minio")
	watchCmd.PersistentFlags().StringP("secretAccessKey", "s", "changeme", "secretAccessKey of minio")
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch pod",
	Long:  `Watch pod`,
	Run: func(cmd *cobra.Command, args []string) {

		var (
			done          = make(chan struct{})
			podStatusChan = make(chan *corev1.Pod)
		)

		dir, _ := cmd.Flags().GetString("dir")
		endpoint, _ := cmd.Flags().GetString("endpoint")
		accessKeyID, _ := cmd.Flags().GetString("accessKeyID")
		secretAccessKey, _ := cmd.Flags().GetString("secretAccessKey")

		c, _ := minio.Client(endpoint, accessKeyID, secretAccessKey)

		client, err := k8s.Client()
		if err != nil {
			panic(err)
		}
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		go func(t *time.Ticker) {
			for {
				<-t.C
				pod, err := client.CoreV1().Pods("default").Get(context.TODO(), "pytest", metav1.GetOptions{})
				if err != nil {
					log.Fatalf("Failed to get pod status: %v", err)
				}
				log.Printf("Successed to get pod status %v", pod.Status.Phase)
				podStatusChan <- pod
				if pod.Status.Phase == corev1.PodSucceeded {
					close(done)
					return
				}
			}
		}(ticker)

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					log.Println("event:", event)
					if event.Has(fsnotify.Write) {
						log.Println("Writeite file:", event.Name)
						minio.Upload(c, "result", event.Name)
					}
					if event.Has(fsnotify.Remove) {
						log.Println("Remove file:", event.Name)
					}
					if event.Has(fsnotify.Rename) {
						log.Println("Rename file:", event.Name)
					}
					if event.Has(fsnotify.Create) {
						log.Println("Create file:", event.Name)
						minio.Upload(c, "result", event.Name)
					}
					if event.Has(fsnotify.Chmod) {
						log.Println("Chmod file:", event.Name)
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("error:", err)
				case podStatus := <-podStatusChan:
					if podStatus.Status.ContainerStatuses[0].State.Terminated.Reason == "Completed" {
						break
					}
					log.Printf("Received pod status: %v", podStatus.Status.Phase)
				}
			}
		}()

		err = watcher.Add(dir)
		if err != nil {
			log.Fatalf("Failed to watch directory: %v", err)
		}

		<-done
		log.Println("Pod has succeeded, exiting program.")
		os.Exit(0)
	},
}
