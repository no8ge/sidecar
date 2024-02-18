package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/no8ge/sidecar/pkg/k8s"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	useSSL  = false
	objects = make(chan string)
	done    = make(chan struct{})
)

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.PersistentFlags().StringP("dir", "d", "/data", "the dir for sidecar watch")
	watchCmd.PersistentFlags().StringP("prefix", "p", "pytest", "the prefix for minio")
	watchCmd.PersistentFlags().StringP("namespace", "n", "default", "the namespace")
	watchCmd.PersistentFlags().StringP("bucket", "b", "result", "the bucket of minio")
	watchCmd.PersistentFlags().StringP("endpoint", "e", "files-minio.atop-system.svc:9000", "endpoint of minio")
	watchCmd.PersistentFlags().StringP("accessKeyID", "a", "admin", "accessKeyID of minio")
	watchCmd.PersistentFlags().StringP("secretAccessKey", "s", "changeme", "secretAccessKey of minio")
}

func checkContainerStatus(t *time.Ticker, client *kubernetes.Clientset, namespace string, podname string) {
	for {
		<-t.C
		pod, err := client.CoreV1().Pods(namespace).Get(context.TODO(), podname, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Failed to get pod status: %v", err)
		}
		for _, containerStatus := range pod.Status.ContainerStatuses {
			log.Printf("Successed to get %s status %v", containerStatus.Name, containerStatus.State)
			if containerStatus.Name != "sidecar" && containerStatus.State.Terminated != nil && len(objects) == 0 {
				close(done)
			}
		}
	}
}

func uploadToMinio(ch chan string, minioClient *minio.Client, bucketName string, prefix string) {
	for obj := range ch {
		contentType := "application/octet-stream"
		file, err := os.Open(obj)
		if err != nil {
			log.Println(err)
			return
		}
		defer file.Close()

		fileStat, err := file.Stat()
		if err != nil {
			log.Println(err)
			return
		}
		uploadInfo, err := minioClient.PutObject(context.Background(), bucketName, prefix+obj, file, fileStat.Size(), minio.PutObjectOptions{ContentType: contentType})
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("Successed upload bytes: ", uploadInfo)
	}
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch pod",
	Long:  `Watch pod`,
	Run: func(cmd *cobra.Command, args []string) {

		dir, _ := cmd.Flags().GetString("dir")
		prefix, _ := cmd.Flags().GetString("prefix")
		bucket, _ := cmd.Flags().GetString("bucket")
		endpoint, _ := cmd.Flags().GetString("endpoint")
		namespace, _ := cmd.Flags().GetString("namespace")
		accessKeyID, _ := cmd.Flags().GetString("accessKeyID")
		secretAccessKey, _ := cmd.Flags().GetString("secretAccessKey")

		minioClient, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: useSSL,
		})
		if err != nil {
			log.Fatalln(err)
		}

		k8sClient, err := k8s.Client()
		if err != nil {
			panic(err)
		}
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		go checkContainerStatus(ticker, k8sClient, namespace, prefix)
		go uploadToMinio(objects, minioClient, bucket, prefix)

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
					if event.Has(fsnotify.Write) {
						log.Println("Successed to write file:", event.Name)
						objects <- event.Name
					}
					if event.Has(fsnotify.Remove) {
						log.Println("Successed to remove file:", event.Name)
					}
					if event.Has(fsnotify.Rename) {
						log.Println("Successed to rename file:", event.Name)
					}
					if event.Has(fsnotify.Create) {
						fileInfo, err := os.Stat(event.Name)
						if err != nil {
							log.Println("Failed to stat file", event.Name)
						}
						if fileInfo.IsDir() {
							watcher.Add(event.Name)
							if err != nil {
								log.Fatalf("Failed to watch directory: %v", err)
							}
							log.Println("Successed to create dir:", event.Name)
						}
						if !fileInfo.IsDir() {
							log.Println("Successed to create file:", event.Name)
							objects <- event.Name
						}
					}
					if event.Has(fsnotify.Chmod) {
						log.Println("Successed to chmod file:", event.Name)
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("error:", err)
				}
			}
		}()

		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				log.Println("Successed to find directory:", path)
				err = watcher.Add(path)
				if err != nil {
					log.Fatalf("Failed to watch directory: %v", err)
				}
			} else {
				log.Println("Successed to find file:", path)
				objects <- path
			}
			return nil
		})
		if err != nil {
			fmt.Println("Error:", err)
		}

		<-done
		log.Println("Pod has succeeded, exiting program.")
		os.Exit(0)
	},
}
