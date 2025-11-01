# Jenkins Configuration for Zoraxy

This directory contains Jenkins configuration files and documentation for setting up automated builds for the Zoraxy project.

## Files

### Pipeline Files (Project Root)
- **[`Jenkinsfile`](../Jenkinsfile)** - Standard Jenkins pipeline for basic builds
- **[`Jenkinsfile.advanced`](../Jenkinsfile.advanced)** - Advanced pipeline with additional features
- **[`Jenkinsfile.docker`](../Jenkinsfile.docker)** - Docker-focused pipeline

### Configuration Files
- **[`build-config.yml`](build-config.yml)** - Build configuration parameters
- **[`jenkins-job-config.xml`](jenkins-job-config.xml)** - Jenkins job XML template

### Documentation
- **[`setup-guide.md`](setup-guide.md)** - Detailed setup instructions
- **[`QUICK-REFERENCE.md`](QUICK-REFERENCE.md)** - Quick reference and cheat sheet
- **[`FAQ.md`](FAQ.md)** - Frequently asked questions

### Utilities
- **[`validate-jenkinsfiles.sh`](validate-jenkinsfiles.sh)** - Validation script for Jenkinsfiles
- **[`examples/`](examples/)** - Example pipeline configurations

## Quick Setup

### Prerequisites

1. Jenkins server (version 2.300 or later)
2. Required Jenkins plugins:
   - Pipeline
   - Git
   - Docker Pipeline (if building Docker images)
   - Email Extension
   - Workspace Cleanup

### Basic Setup Steps

1. **Install Required Plugins**
   - Navigate to: Jenkins → Manage Jenkins → Manage Plugins
   - Install the required plugins listed above

2. **Create New Pipeline Job**
   - Click "New Item"
   - Enter job name: "Zoraxy-Build"
   - Select "Pipeline"
   - Click "OK"

3. **Configure Pipeline**
   - In the Pipeline section, set:
     - Definition: "Pipeline script from SCM"
     - SCM: Git
     - Repository URL: Your Zoraxy repository URL
     - Script Path: `Jenkinsfile` (or `Jenkinsfile.advanced`)

4. **Configure Build Triggers** (Optional)
   - Poll SCM: `H/15 * * * *` (check every 15 minutes)
   - Or use GitHub webhooks for immediate builds

5. **Save and Build**
   - Save the configuration
   - Click "Build with Parameters" to start your first build

## Pipeline Features

### Standard Pipeline (Jenkinsfile)

- Multi-platform binary builds (Linux, Windows, macOS, FreeBSD)
- Docker image building
- Checksum generation
- Artifact archiving
- Email notifications

### Advanced Pipeline (Jenkinsfile.advanced)

All features from the standard pipeline, plus:
- Code quality checks (linting, formatting)
- Unit tests with coverage reports
- Security vulnerability scanning
- Parallel build stages
- Release package creation
- Enhanced error handling
- Build metrics and reporting

## Build Parameters

### Standard Parameters

- **BUILD_TYPE**: Select release or development build
- **BUILD_DOCKER**: Enable/disable Docker image building
- **RUN_TESTS**: Enable/disable test execution

### Advanced Parameters

- **BUILD_TYPE**: release, development, or nightly
- **VERSION_TAG**: Override automatic version tagging
- **BUILD_ALL_PLATFORMS**: Build for all platforms or just Linux AMD64
- **BUILD_DOCKER**: Enable/disable Docker builds
- **RUN_SECURITY_SCAN**: Enable security scanning

## Environment Variables

Configure these in Jenkins:

- `DOCKER_REGISTRY` - Docker registry credentials (for pushing images)
- `DEFAULT_RECIPIENTS` - Email addresses for build notifications

## Troubleshooting

### Build Fails with "go: command not found"

Ensure Go is installed on your Jenkins agent. You can:
1. Install Go directly on the agent
2. Use a Docker agent with Go pre-installed
3. Add Go installation step to the pipeline

### Docker Build Fails

1. Verify Docker is installed and running on the agent
2. Check that the Jenkins user has Docker permissions
3. Verify Docker registry credentials are configured

### Tests Fail

Check the test output in the build console. Common issues:
- Missing dependencies
- Network connectivity issues
- Platform-specific test failures

## Customization

### Adding New Platforms

Edit the Makefile or add new build stages in the Jenkinsfile:

```groovy
stage('Build New Platform') {
    steps {
        sh '''
            cd ${BUILD_DIR}
            GOOS=newos GOARCH=newarch CGO_ENABLED=0 go build -o dist/zoraxy_newos_newarch -ldflags "-s -w" -trimpath
        '''
    }
}
```

### Modifying Build Flags

Update the `go build` command with your desired flags:
- `-ldflags "-s -w"` - Strip debug information (smaller binaries)
- `-trimpath` - Remove local path information
- `-race` - Enable race detector (testing only)

## Support

For issues or questions:
- Check the main project README
- Review Jenkins console output
- Check Jenkins system logs

## License

This configuration is part of the Zoraxy project and follows the same AGPL license.
