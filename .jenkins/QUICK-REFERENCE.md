# Jenkins Quick Reference for Zoraxy

A cheat sheet for common Jenkins tasks and configurations for the Zoraxy project.

## Quick Links

- [Jenkinsfile](../Jenkinsfile) - Standard pipeline
- [Jenkinsfile.advanced](../Jenkinsfile.advanced) - Advanced pipeline  
- [Jenkinsfile.docker](../Jenkinsfile.docker) - Docker builds
- [Setup Guide](setup-guide.md) - Detailed setup instructions

## Common Commands

### Start a Build

```bash
# Via Jenkins CLI
java -jar jenkins-cli.jar -s http://localhost:8080/ build Zoraxy-Build

# With parameters
java -jar jenkins-cli.jar -s http://localhost:8080/ build Zoraxy-Build \
  -p BUILD_TYPE=release \
  -p BUILD_DOCKER=true
```

### Check Build Status

```bash
# Get last build status
java -jar jenkins-cli.jar -s http://localhost:8080/ \
  get-build Zoraxy-Build lastBuild
```

### Download Artifacts

```bash
# Download artifacts from last successful build
wget http://localhost:8080/job/Zoraxy-Build/lastSuccessfulBuild/artifact/src/dist/zoraxy_linux_amd64
```

## Pipeline Configuration Matrix

| Feature | Jenkinsfile | Jenkinsfile.advanced | Jenkinsfile.docker |
|---------|-------------|----------------------|--------------------|
| Multi-platform builds | ✓ | ✓ | - |
| Docker builds | ✓ | ✓ | ✓ |
| Code linting | - | ✓ | - |
| Unit tests | ✓ | ✓ | - |
| Security scanning | - | ✓ | - |
| Coverage reports | - | ✓ | - |
| Multi-arch Docker | - | - | ✓ |
| Docker registry push | Optional | Optional | ✓ |
| Email notifications | ✓ | ✓ | ✓ |
| Artifact archiving | ✓ | ✓ | ✓ |

## Build Parameters Quick Reference

### Standard Jenkinsfile

```groovy
BUILD_TYPE: 'release' or 'development'
BUILD_DOCKER: true/false
RUN_TESTS: true/false
```

### Advanced Jenkinsfile

```groovy
BUILD_TYPE: 'release', 'development', or 'nightly'
VERSION_TAG: 'v1.0.0' (optional, auto if empty)
BUILD_ALL_PLATFORMS: true/false
BUILD_DOCKER: true/false
RUN_SECURITY_SCAN: true/false
```

### Docker Jenkinsfile

```groovy
DOCKER_TAG: 'latest' (or custom tag)
PUSH_TO_REGISTRY: true/false
PLATFORMS: 'linux/amd64,linux/arm64' (or subset)
```

## Platform Build Targets

| Platform | GOOS | GOARCH | GOARM | Output File |
|----------|------|--------|-------|-------------|
| Linux AMD64 | linux | amd64 | - | zoraxy_linux_amd64 |
| Linux ARM64 | linux | arm64 | - | zoraxy_linux_arm64 |
| Linux ARM | linux | arm | 6 | zoraxy_linux_arm |
| Linux 386 | linux | 386 | - | zoraxy_linux_386 |
| Windows AMD64 | windows | amd64 | - | zoraxy_windows_amd64.exe |
| macOS AMD64 | darwin | amd64 | - | zoraxy_darwin_amd64 |
| FreeBSD AMD64 | freebsd | amd64 | - | zoraxy_freebsd_amd64 |

## Build Flags Reference

```bash
# Standard build flags used in all Jenkinsfiles
GOOS=linux              # Target OS
GOARCH=amd64            # Target architecture  
CGO_ENABLED=0           # Disable CGO for static binary
-ldflags "-s -w"        # Strip debug info (reduces size)
-trimpath               # Remove local path info
```

## Environment Variables

Configure these in Jenkins:

```bash
GO_VERSION=1.24                 # Go version
PROJECT_NAME=zoraxy             # Project name
BUILD_DIR=src                   # Source directory
DIST_DIR=src/dist               # Output directory
DOCKER_REGISTRY=docker.io       # Docker registry
```

## Cron Syntax Quick Reference

```bash
# Poll SCM every 15 minutes
H/15 * * * *

# Daily at 2 AM
H 2 * * *

# Every Sunday at midnight
H 0 * * 0

# Every hour
H * * * *

# Weekdays at 6 PM
H 18 * * 1-5
```

## Useful Jenkins CLI Commands

```bash
# Download Jenkins CLI
wget http://localhost:8080/jnlpJars/jenkins-cli.jar

# List all jobs
java -jar jenkins-cli.jar -s http://localhost:8080/ list-jobs

# Get job configuration
java -jar jenkins-cli.jar -s http://localhost:8080/ get-job Zoraxy-Build

# Create job from XML
java -jar jenkins-cli.jar -s http://localhost:8080/ create-job Zoraxy-Build \
  < .jenkins/jenkins-job-config.xml

# Delete job
java -jar jenkins-cli.jar -s http://localhost:8080/ delete-job Zoraxy-Build

# Console output
java -jar jenkins-cli.jar -s http://localhost:8080/ console Zoraxy-Build

# Build with parameters
java -jar jenkins-cli.jar -s http://localhost:8080/ build Zoraxy-Build \
  -p BUILD_TYPE=release -p BUILD_DOCKER=true -s
```

## Git Hooks for Automatic Builds

### Post-commit hook

Create `.git/hooks/post-commit`:

```bash
#!/bin/bash
# Trigger Jenkins build after commit
curl -X POST http://localhost:8080/job/Zoraxy-Build/build
```

### Pre-push validation

Create `.git/hooks/pre-push`:

```bash
#!/bin/bash
# Validate before push
cd src
go test ./...
go vet ./...
```

## Docker Commands

```bash
# Build Docker image locally
cd docker
docker build -t zoraxy:test .

# Run container
docker run -d -p 8000:8000 zoraxy:test

# Check logs
docker logs -f <container_id>

# Multi-arch build
docker buildx build --platform linux/amd64,linux/arm64 -t zoraxy:latest .
```

## Debugging Tips

### View Build Logs

```bash
# Via web
http://localhost:8080/job/Zoraxy-Build/lastBuild/console

# Via CLI
java -jar jenkins-cli.jar -s http://localhost:8080/ \
  console Zoraxy-Build -f
```

### Check Workspace

```bash
# List workspace files
ls -la /var/lib/jenkins/workspace/Zoraxy-Build/

# View specific file
cat /var/lib/jenkins/workspace/Zoraxy-Build/src/go.mod
```

### Pipeline Replay

1. Go to build page
2. Click "Replay" in left menu
3. Modify pipeline script
4. Click "Run" to test changes

### Test Go Build Locally

```bash
cd src
go mod tidy
GOOS=linux GOARCH=amd64 go build -o test_binary
./test_binary -version
```

## Common Issues and Solutions

### Issue: "go: command not found"

**Solution:** Add Go to PATH in Jenkins configuration

```groovy
environment {
    PATH = "/usr/local/go/bin:${env.PATH}"
}
```

### Issue: Docker permission denied

**Solution:** Add jenkins user to docker group

```bash
sudo usermod -aG docker jenkins
sudo systemctl restart jenkins
```

### Issue: Workspace disk full

**Solution:** Enable workspace cleanup

```groovy
options {
    buildDiscarder(logRotator(numToKeepStr: '10'))
}

post {
    always {
        cleanWs()
    }
}
```

### Issue: Build timeout

**Solution:** Increase timeout

```groovy
options {
    timeout(time: 2, unit: 'HOURS')
}
```

## Performance Tips

1. **Cache Go modules**: Use shared Go module cache
2. **Parallel builds**: Build different platforms in parallel
3. **Incremental builds**: Use `skipDefaultCheckout()`
4. **Docker layer caching**: Use buildx cache
5. **Workspace cleanup**: Always clean after build
6. **Artifact retention**: Limit old builds and artifacts

## Security Checklist

- [ ] Use credentials plugin for secrets
- [ ] Enable HTTPS for Jenkins
- [ ] Configure CSRF protection
- [ ] Use matrix-based security
- [ ] Regular plugin updates
- [ ] Audit trail enabled
- [ ] Secure webhook endpoints
- [ ] Limit executor access
- [ ] Use SSH for Git access
- [ ] Scan dependencies regularly

## Monitoring

### Key Metrics to Track

- Build success rate
- Average build duration
- Queue time
- Workspace disk usage
- Failed tests
- Code coverage trends
- Artifact sizes

### Jenkins Plugins for Monitoring

- Build Monitor View
- Build Failure Analyzer
- Metrics Plugin
- Disk Usage Plugin
- Build Time Analyzer

## Additional Resources

- 📖 [Jenkins Documentation](https://www.jenkins.io/doc/)
- 🎓 [Pipeline Tutorial](https://www.jenkins.io/doc/book/pipeline/)
- 💬 [Jenkins Community](https://community.jenkins.io/)
- 🐛 [Issue Tracker](https://issues.jenkins.io/)
- 📚 [Plugin Index](https://plugins.jenkins.io/)

---

**Quick help:** For issues, check the [Setup Guide](setup-guide.md) or [Troubleshooting section](setup-guide.md#troubleshooting)
