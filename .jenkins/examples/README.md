# Jenkins Pipeline Examples

This directory contains example Jenkins pipeline configurations for various use cases.

## Examples

### 1. simple-pipeline.groovy

A minimal Jenkinsfile for getting started quickly.

**Features:**
- Basic checkout and build
- Single Linux AMD64 binary
- Artifact archiving

**Usage:**
Copy this file to your project root as `Jenkinsfile` to get started.

### 2. Using the Standard Jenkinsfile

The main `Jenkinsfile` in the project root provides:
- Multi-platform builds
- Docker image building
- Test execution
- Checksum generation
- Email notifications

### 3. Using Jenkinsfile.advanced

For production environments with additional features:
- Code quality checks (linting, formatting)
- Unit tests with coverage
- Security vulnerability scanning
- Parallel builds
- Release packaging
- Enhanced reporting

### 4. Using Jenkinsfile.docker

For Docker-focused workflows:
- Multi-architecture Docker builds
- Push to Docker registry
- Image testing
- Manifest generation

## Customization Tips

### Building for Specific Platforms

Edit the `Build Binaries` stage in the Jenkinsfile:

```groovy
stage('Build Linux AMD64') {
    steps {
        sh '''
            cd src
            GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/zoraxy_linux_amd64
        '''
    }
}
```

### Adding Custom Build Steps

Add a new stage after the build:

```groovy
stage('Custom Step') {
    steps {
        echo 'Running custom step...'
        sh 'your-custom-command'
    }
}
```

### Conditional Execution

Run stages conditionally:

```groovy
stage('Deploy') {
    when {
        branch 'main'
    }
    steps {
        echo 'Deploying...'
    }
}
```

### Environment-Specific Configuration

Use parameters for different environments:

```groovy
parameters {
    choice(name: 'ENVIRONMENT', choices: ['dev', 'staging', 'prod'])
}

stage('Deploy') {
    steps {
        script {
            if (params.ENVIRONMENT == 'prod') {
                // Production deployment
            }
        }
    }
}
```

## Testing Your Pipeline

### Local Testing

Use the validation script:

```bash
.jenkins/validate-jenkinsfiles.sh
```

### Jenkins Replay

1. Run a build
2. Click "Replay" in the build menu
3. Edit the pipeline script
4. Click "Run"

This allows testing changes without committing.

## Best Practices

1. **Start Simple**: Use `simple-pipeline.groovy` for initial setup
2. **Iterate**: Add features gradually as needed
3. **Test Changes**: Use Jenkins Replay to test before committing
4. **Use Parameters**: Make your pipeline configurable
5. **Handle Failures**: Always include proper error handling
6. **Clean Up**: Use `cleanWs()` to avoid disk space issues
7. **Secure Credentials**: Never hardcode secrets in Jenkinsfile

## Additional Resources

- [Jenkins Pipeline Syntax](https://www.jenkins.io/doc/book/pipeline/syntax/)
- [Pipeline Steps Reference](https://www.jenkins.io/doc/pipeline/steps/)
- [Groovy Documentation](https://groovy-lang.org/documentation.html)
