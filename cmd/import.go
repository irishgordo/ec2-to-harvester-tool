/*
Stuff

*/
package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultCommand = "qemu-wrapper.sh"

func ConvertVMDKtoRAW(source, target string) error {
	args := []string{"convert", "-f", "vmdk", "-O", "raw", source, target}
	cmd := exec.Command(defaultCommand, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	return cmd.Run()
}

// BucketBasics encapsulates the Amazon Simple Storage Service (Amazon S3) actions
// used in the examples.
// It contains S3Client, an Amazon S3 service client that is used to perform bucket
// and object actions.
type BucketBasics struct {
	S3Client *s3.Client
}

// DownloadFile gets an object from a bucket and stores it in a local file.
func (basics BucketBasics) DownloadFile(bucketName string, objectKey string, fileName string) error {
	result, err := basics.S3Client.GetObject(rootCmd.Context(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		log.Printf("Couldn't get object %v:%v. Here's why: %v\n", bucketName, objectKey, err)
		return err
	}
	defer result.Body.Close()
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("Couldn't create file %v. Here's why: %v\n", fileName, err)
		return err
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("Couldn't read object body from %v. Here's why: %v\n", objectKey, err)
	}
	_, err = file.Write(body)
	return err
}

func ensureDir(dirName string) error {
	err := os.Mkdir(dirName, 0777)
	if err == nil {
		return nil
	}
	if os.IsExist(err) {
		// check that the existing path is a directory
		info, err := os.Stat(dirName)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return errors.New("path exists but is not a directory")
		}
		return nil
	}
	return err
}

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "import ec2 instance to harvester via s3",
	Long:  `import ec2 instance via s3 to harvester`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("import called")
		ec2Id := viper.GetString("ec2-instance-id")
		awsRegion := viper.GetString("aws-region")
		s3BucketId := viper.GetString("s3-bucket-name")
		fmt.Printf("ec2-instance-id being used... : %#v\n", ec2Id)
		fmt.Printf("aws-region being used... : %#v\n", awsRegion)
		fmt.Printf("s3-bucket-id being used... : %#v\n", s3BucketId)

		config, err := config.LoadDefaultConfig(cmd.Context(), config.WithRegion(awsRegion))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ec2Service := ec2.NewFromConfig(config)

		result, err := ec2Service.DescribeInstances(cmd.Context(), &ec2.DescribeInstancesInput{
			Filters: []ec2Types.Filter{
				{
					Name: aws.String("instance-state-name"),
					Values: []string{
						"running",
						"pending",
						"stopped",
					},
				},
				{
					Name: aws.String("architecture"),
					Values: []string{
						"x86_64",
					},
				},
				{
					Name: aws.String("instance-id"),
					Values: []string{
						ec2Id,
					},
				},
			},
		})
		if err != nil {
			fmt.Printf("Need to do better error handling...\n But we encountered an error with the fetching of the EC2 Instance...")
			return
		}
		fmt.Printf("%#v\n", result)
		ec2CpuCoreCount := (*result).Reservations[0].Instances[0].CpuOptions.CoreCount
		fmt.Printf("cpu-core-count: %#v\n", ec2CpuCoreCount)
		ec2Type := (*result).Reservations[0].Instances[0].InstanceType
		fmt.Printf("ec2-instance-type: %#v\n", ec2Type)
		s3Service := s3.NewFromConfig(config)
		s3Result, err := s3Service.GetBucketLocation(cmd.Context(), &s3.GetBucketLocationInput{
			Bucket: aws.String(s3BucketId),
		})
		if err != nil {
			fmt.Printf("Need to do better error handling...\n But we saw something at the S3 fetching that was weird...")
			return
		}
		fmt.Printf("bucket info: %#v", s3Result)
		s3Location := string((*s3Result).LocationConstraint)
		if s3Location != awsRegion {
			fmt.Printf("Need better error handling... Bucket isn't in the same region however...")
			return
		}
		ec2ExportTask, err := ec2Service.CreateInstanceExportTask(cmd.Context(), &ec2.CreateInstanceExportTaskInput{
			Description: aws.String("ec2 to harvester tool export task..."),
			InstanceId:  aws.String(ec2Id),
			ExportToS3Task: &ec2Types.ExportToS3TaskSpecification{
				DiskImageFormat: ec2Types.DiskImageFormatVmdk,
				S3Bucket:        aws.String(s3BucketId),
			},
			TargetEnvironment: ec2Types.ExportEnvironmentVmware,
		})
		if err != nil {
			fmt.Printf("issue with creating the ec2ExportTask, not sure, hmmm....")
			return
		}
		fmt.Printf("ec2 export task: %#v\n", ec2ExportTask)
		newExportTaskId := (*ec2ExportTask.ExportTask).ExportTaskId
		exportTaskIdStr := aws.String(string(*newExportTaskId))
		ec2ExportTaskOutput, err := ec2Service.DescribeExportTasks(cmd.Context(), &ec2.DescribeExportTasksInput{
			ExportTaskIds: []string{
				*exportTaskIdStr,
			},
		})
		if err != nil {
			fmt.Printf("issue tracking export")
			return
		}
		fmt.Printf("exportTaskIdOutput: %#v \n", ec2ExportTaskOutput)
		currentExportState := ec2ExportTaskOutput.ExportTasks[0].State
		currentExportStateStr := string(currentExportState)
		fmt.Printf("current-export-state: %#v\n", currentExportStateStr)
		i := 1
		max := 30
		for i < max {
			fmt.Printf("Current Unix Time: %v\n", time.Now().Unix())
			fmt.Println(i)
			i += 1
			nestedEc2ExportTaskOutput, err := ec2Service.DescribeExportTasks(cmd.Context(), &ec2.DescribeExportTasksInput{
				ExportTaskIds: []string{
					*exportTaskIdStr,
				},
			})
			if err != nil {
				fmt.Printf("issue tracking export")
				return
			}
			fmt.Printf("exportTaskIdOutput: %#v \n", nestedEc2ExportTaskOutput)
			currentExportState = nestedEc2ExportTaskOutput.ExportTasks[0].State
			currentExportStateStr = string(currentExportState)
			fmt.Printf("current-export-state: %#v\n", currentExportStateStr)
			if currentExportStateStr == "completed" {
				s3Key := (*(*nestedEc2ExportTaskOutput).ExportTasks[0].ExportToS3Task).S3Key
				s3Bucket := (*(*(*nestedEc2ExportTaskOutput).ExportTasks[0].ExportToS3Task).S3Bucket)
				result, err := s3Service.GetObject(rootCmd.Context(), &s3.GetObjectInput{
					Bucket: aws.String(s3Bucket),
					Key:    aws.String(*s3Key),
				})
				if err != nil {
					fmt.Printf("issue getting s3 file")
					return
				}
				defer result.Body.Close()
				err = ensureDir("/tmp/ec2-to-harvester")
				if err != nil {
					fmt.Println("issue with building directory in tmp...")
					return
				}
				file, err := os.Create("/tmp/ec2-to-harvester/" + s3Bucket + "_" + *s3Key)
				if err != nil {
					fmt.Printf("issue with writing file...")
					return
				}
				defer file.Close()
				body, err := io.ReadAll(result.Body)
				if err != nil {
					fmt.Println("issue with reading file...")
					return
				}
				_, err = file.Write(body)
				if err != nil {
					fmt.Println("issue writing file...")
					return
				}
				curFile := filepath.Join("/tmp/ec2-to-harvester/", s3Bucket+"_"+*s3Key)
				destFile := filepath.Join("/tmp/ec2-to-harvester/", s3Bucket+"_"+*s3Key+".img")
				err = ConvertVMDKtoRAW(curFile, destFile)
				if err != nil {
					fmt.Println("issue converting the file from vmdk to raw...")
					return
				}
				fmt.Printf("\nFinished Successful downloading at:\n%#v\n", destFile)
				break
			} else {
				fmt.Println("sleeping for one minute then polling again...")
				time.Sleep(60 * time.Second)
				fmt.Println("finished sleep...")
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	importCmd.PersistentFlags().String("ec2-instance-id", "", "an ec2 instance id")
	importCmd.PersistentFlags().String("aws-region", "", "aws region for instance and bucket")
	importCmd.PersistentFlags().String("s3-bucket-name", "", "s3 bucket name")
	importCmd.MarkPersistentFlagRequired("ec2-instance-id")
	importCmd.MarkPersistentFlagRequired("aws-region")
	importCmd.MarkPersistentFlagRequired("s3-bucket-name")

	viper.BindPFlag("ec2-instance-id", importCmd.PersistentFlags().Lookup("ec2-instance-id"))
	viper.BindPFlag("aws-region", importCmd.PersistentFlags().Lookup("aws-region"))
	viper.BindPFlag("s3-bucket-name", importCmd.PersistentFlags().Lookup("s3-bucket-name"))

}
