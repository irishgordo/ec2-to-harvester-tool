## TODO


#### Sample launch.json:
```json
{
    "version": "0.2.0",
    "configurations": [
      {
        "name": "Launch file",
        "type": "go",
        "request": "launch",
        "mode": "debug",
        "program": "${workspaceFolder}/main.go",
        "args": ["import", "--aws-region", "us-west-2", "--ec2-instance-id", "i-abcdefg12345", "--s3-bucket-name", "bucketnameinaws"]
      },
    ]
  }
```

```
"Bucket must be owned by the same S3 account and READ_ACL and WRITE permissions are required on the destination bucket. Please refer to the API documentation for details."
```
Needs region specific cannonical id:
https://docs.aws.amazon.com/vm-import/latest/userguide/vmexport.html

```
"This instance has multiple volumes attached. Please remove additional volumes."
```
( be aware this AWS can only export single volume (root disk) based VMs )

Needs custom `qemu-wrapper.sh` in path:
```
╭─mike at suse-workstation-team-harvester in ~
╰─○ history | grep -ie "qemu-wrapper.sh"
 6145  10/26/2022 15:53  nvim ~/.local/bin/qemu-wrapper.sh
 6147  10/26/2022 15:53  which qemu-wrapper.sh
 6148  10/26/2022 15:53  chmod +x ~/.local/bin/qemu-wrapper.sh
 6150  10/26/2022 15:54  which qemu-wrapper.sh
 6153  10/26/2022 16:00  which qemu-wrapper.sh
 6156  10/26/2022 16:01  which qemu-wrapper.sh
 6195  10/27/2022 10:14  which qemu-wrapper.sh
╭─mike at suse-workstation-team-harvester in ~
╰─○ cat .local/bin/qemu-wrapper.sh
#!/bin/bash
ulimit -v 1048576
qemu-img "$@"
╭─mike at suse-workstation-team-harvester in ~
╰─○ which bash
/usr/bin/bash
╭─mike at suse-workstation-team-harvester in ~
╰─○ qemu-img --version
qemu-img version 6.2.0 (Debian 1:6.2+dfsg-2ubuntu6.6)
Copyright (c) 2003-2021 Fabrice Bellard and the QEMU Project developers

```
File with executable like:
```
#!/bin/bash
ulimit -v 1048576
qemu-img "$@"
```

### Requires
- aws-cli set up