# Jenkins FAQ for Zoraxy

Frequently Asked Questions about setting up and using Jenkins with Zoraxy.

## General Questions

### Q: Which Jenkinsfile should I use?

**A:** Choose based on your needs:
- **Jenkinsfile**: Best for most users. Builds all platforms and includes Docker support.
- **Jenkinsfile.advanced**: For teams needing code quality checks, testing, and security scanning.
- **Jenkinsfile.docker**: When you only need Docker images (no platform-specific binaries).
- **examples/simple-pipeline.groovy**: Learning or very basic needs (single platform).

### Q: What are the minimum system requirements?

**A:** 
- **Jenkins Server**: 2GB RAM, 2 CPU cores, 10GB disk space
- **Go**: Version 1.23 or higher
- **Java**: JDK 11 or 17 (for Jenkins itself)
- **Docker**: Latest version (if building containers)

### Q: Can I run Jenkins in a container?

**A:** Yes! Use the official Jenkins Docker image:

```bash
docker run -d \
  --name jenkins \
  -p 8080:8080 -p 50000:50000 \
  -v jenkins_home:/var/jenkins_home \
  -v /var/run/docker.sock:/var/run/docker.sock \
  jenkins/jenkins:lts
```

### Q: How do I update to a newer version of Go?

**A:** 
1. Install the new Go version on your Jenkins agent
2. Update the `GO_VERSION` environment variable in your Jenkinsfile
3. Or configure it globally in Jenkins → Manage Jenkins → Global Tool Configuration

## Installation and Setup

### Q: Jenkins fails to start. What should I check?

**A:** Common issues:
1. Port 8080 already in use: Change port with `--httpPort=9090`
2. Insufficient permissions: Run as proper user or with sudo
3. Java not installed: Install JDK 11 or 17
4. Check logs: `sudo journalctl -u jenkins -f`

### Q: How do I install Jenkins on Ubuntu?

**A:** Follow these steps:

```bash
curl -fsSL https://pkg.jenkins.io/debian-stable/jenkins.io-2023.key | sudo tee \
  /usr/share/keyrings/jenkins-keyring.asc > /dev/null

echo deb [signed-by=/usr/share/keyrings/jenkins-keyring.asc] \
  https://pkg.jenkins.io/debian-stable binary/ | sudo tee \
  /etc/apt/sources.list.d/jenkins.list > /dev/null

sudo apt-get update
sudo apt-get install fontconfig openjdk-17-jre jenkins
sudo systemctl start jenkins
```

### Q: Where do I find the initial admin password?

**A:** 

```bash
sudo cat /var/lib/jenkins/secrets/initialAdminPassword
```

### Q: Which Jenkins plugins are essential?

**A:** Required plugins:
- Pipeline
- Git Plugin
- Workspace Cleanup
- Email Extension
- Credentials Plugin

Recommended:
- Docker Pipeline (for Docker builds)
- Blue Ocean (better UI)
- Build Monitor View
- AnsiColor (colored output)

## Configuration

### Q: How do I configure credentials for Docker Hub?

**A:**
1. Go to Jenkins → Manage Jenkins → Manage Credentials
2. Click "Add Credentials"
3. Select "Username with password"
4. Enter Docker Hub username and password/token
5. Set ID as `docker-registry-credentials`
6. Save

### Q: How do I set up email notifications?

**A:**
1. Go to Jenkins → Manage Jenkins → Configure System
2. Find "Extended E-mail Notification"
3. Configure SMTP server (e.g., smtp.gmail.com:465)
4. Add credentials (use app password for Gmail)
5. Enable SSL
6. Test configuration

For Gmail:
- Enable 2FA
- Create app-specific password
- Use that password in Jenkins

### Q: How do I enable GitHub webhooks?

**A:**
1. In GitHub repo: Settings → Webhooks → Add webhook
2. Payload URL: `http://your-jenkins:8080/github-webhook/`
3. Content type: `application/json`
4. Select "Just the push event"
5. In Jenkins job: Enable "GitHub hook trigger for GITScm polling"

### Q: Can I use a private Git repository?

**A:** Yes:
1. Jenkins → Manage Jenkins → Manage Credentials
2. Add SSH key or username/password
3. In pipeline configuration, select these credentials for SCM

## Building and Testing

### Q: Build fails with "go: command not found"

**A:** Solutions:
1. Ensure Go is installed on Jenkins agent
2. Add Go to PATH in Jenkinsfile:
```groovy
environment {
    PATH = "/usr/local/go/bin:${env.PATH}"
}
```
3. Or configure Go tool in Jenkins Global Tool Configuration

### Q: How do I build for specific platforms only?

**A:** Edit the Jenkinsfile and remove unwanted platforms from the parallel stages, or use build parameters:

```groovy
parameters {
    booleanParam(name: 'BUILD_WINDOWS', defaultValue: false)
    booleanParam(name: 'BUILD_MACOS', defaultValue: false)
}

stage('Build Windows') {
    when { expression { params.BUILD_WINDOWS } }
    steps { /* build */ }
}
```

### Q: Tests are failing. How do I debug?

**A:**
1. Check console output in Jenkins
2. Run tests locally: `cd src && go test -v ./...`
3. Add verbose test output in Jenkinsfile: `-v` flag
4. Check if dependencies are properly downloaded
5. Verify environment variables

### Q: Build takes too long. How to speed it up?

**A:** Optimization tips:
1. Use parallel stages for multi-platform builds (already in Jenkinsfile)
2. Enable Go module caching
3. Use Docker layer caching
4. Run tests in parallel
5. Build only changed platforms
6. Use multiple Jenkins agents

### Q: How do I skip tests temporarily?

**A:** Use build parameters:

```bash
# Via CLI
java -jar jenkins-cli.jar build Zoraxy-Build -p RUN_TESTS=false

# Or in Jenkinsfile
when {
    expression { params.RUN_TESTS == true }
}
```

## Docker Builds

### Q: Docker build fails with permission denied

**A:**
```bash
# Add jenkins user to docker group
sudo usermod -aG docker jenkins
sudo systemctl restart jenkins

# Verify
sudo -u jenkins docker ps
```

### Q: How do I build multi-architecture Docker images?

**A:** Use `Jenkinsfile.docker` which includes buildx support, or:

```groovy
sh '''
    docker buildx create --name builder --use
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --push \
        -t myrepo/zoraxy:latest .
'''
```

### Q: Can I test Docker images before pushing?

**A:** Yes, set `PUSH_TO_REGISTRY=false` parameter, or add test stage:

```groovy
stage('Test Docker') {
    steps {
        sh '''
            docker run -d --name test zoraxy:latest
            sleep 10
            curl -f http://localhost:8000 || exit 1
            docker stop test && docker rm test
        '''
    }
}
```

### Q: How do I use a private Docker registry?

**A:**
1. Configure credentials in Jenkins
2. Update `DOCKER_REGISTRY` in Jenkinsfile
3. Login in pipeline:
```groovy
withCredentials([usernamePassword(...)]) {
    sh 'echo $PASS | docker login -u $USER --password-stdin registry.com'
}
```

## Troubleshooting

### Q: Workspace is consuming too much disk space

**A:** Solutions:
1. Enable automatic cleanup:
```groovy
options {
    buildDiscarder(logRotator(numToKeepStr: '10'))
}
post {
    always { cleanWs() }
}
```
2. Manual cleanup:
```bash
cd /var/lib/jenkins/workspace
find . -maxdepth 1 -type d -mtime +30 -exec rm -rf {} \;
```

### Q: Jenkins is slow or unresponsive

**A:** Check:
1. Disk space: `df -h`
2. Memory usage: `free -h`
3. Increase Jenkins heap: Edit `/etc/default/jenkins`, add `-Xmx2048m`
4. Disable unused plugins
5. Reduce number of executors
6. Check for zombie processes

### Q: Build hangs at a certain stage

**A:**
1. Check if stage waits for input (prompts)
2. Increase timeout:
```groovy
options {
    timeout(time: 2, unit: 'HOURS')
}
```
3. Check agent connectivity
4. Review console output for blocked processes

### Q: Can't access Jenkins from outside localhost

**A:**
1. Configure Jenkins URL: Manage Jenkins → Configure System
2. Update firewall rules:
```bash
sudo ufw allow 8080/tcp
```
3. Check `JENKINS_ARGS` in `/etc/default/jenkins`
4. Use reverse proxy (Nginx/Apache) for HTTPS

### Q: Emails are not being sent

**A:** Checklist:
- [ ] SMTP server configured correctly
- [ ] Credentials added and working
- [ ] Test configuration button works
- [ ] Email-ext plugin installed
- [ ] Recipients email address correct
- [ ] Check Jenkins system log for errors
- [ ] Firewall allows SMTP port (587/465)

## Advanced Topics

### Q: How do I run builds on multiple agents?

**A:** Configure agents:
1. Manage Jenkins → Manage Nodes → New Node
2. Configure node with labels (e.g., `linux`, `docker`)
3. In Jenkinsfile:
```groovy
pipeline {
    agent { label 'linux' }
    // or
    stages {
        stage('Build') {
            agent { label 'docker' }
            steps { /* ... */ }
        }
    }
}
```

### Q: Can I use Jenkins for continuous deployment?

**A:** Yes, add deployment stages:

```groovy
stage('Deploy to Staging') {
    when { branch 'develop' }
    steps {
        sh './deploy-staging.sh'
    }
}

stage('Deploy to Production') {
    when { branch 'main' }
    steps {
        input message: 'Deploy to production?'
        sh './deploy-production.sh'
    }
}
```

### Q: How do I integrate with Slack/Discord?

**A:**
1. Install Slack/Discord notification plugin
2. Configure webhook URL in Manage Jenkins
3. Add to Jenkinsfile:
```groovy
post {
    success {
        slackSend channel: '#builds', message: 'Build succeeded!'
    }
}
```

### Q: Can I schedule builds?

**A:** Yes, use triggers:

```groovy
triggers {
    cron('H 2 * * *')  // Daily at 2 AM
}
```

Or use build parameters to create on-demand scheduled builds.

### Q: How do I version my builds?

**A:** Use Git tags or generate version:

```groovy
script {
    env.VERSION = sh(
        script: 'git describe --tags --always',
        returnStdout: true
    ).trim()
}
```

## Security

### Q: How do I secure my Jenkins instance?

**A:** Best practices:
1. Enable security: Configure Global Security
2. Use HTTPS (configure reverse proxy)
3. Matrix-based security with least privilege
4. Regular updates of Jenkins and plugins
5. Audit trail plugin
6. Limit network access
7. Use credentials plugin for secrets
8. Enable CSRF protection

### Q: Should I allow anonymous access?

**A:** No, for production. Only for:
- Local development
- Internal networks with other security
- Build status pages (read-only)

### Q: How do I handle secrets in builds?

**A:** Use Credentials Plugin:

```groovy
withCredentials([
    string(credentialsId: 'api-key', variable: 'API_KEY'),
    usernamePassword(credentialsId: 'db-creds', 
                     usernameVariable: 'DB_USER',
                     passwordVariable: 'DB_PASS')
]) {
    sh 'echo $API_KEY | command'
}
```

Never hardcode secrets in Jenkinsfile!

## Getting Help

### Q: Where can I get more help?

**A:** Resources:
- 📖 [Setup Guide](.jenkins/setup-guide.md)
- 📋 [Quick Reference](.jenkins/QUICK-REFERENCE.md)
- 💬 [Jenkins Community](https://community.jenkins.io/)
- 📚 [Official Docs](https://www.jenkins.io/doc/)
- 🐛 [Zoraxy Issues](https://github.com/tobychui/zoraxy/issues)

### Q: How do I report a bug with the Jenkins configuration?

**A:**
1. Check if it's already reported
2. Gather information:
   - Jenkins version
   - Plugin versions
   - Console output
   - Jenkinsfile being used
3. Create issue on GitHub with details

### Q: Can I contribute improvements to the Jenkins setup?

**A:** Absolutely! 
1. Fork the repository
2. Make improvements
3. Test thoroughly
4. Submit pull request
5. Include documentation updates

---

**Still have questions?** Check the [Setup Guide](setup-guide.md) or ask in the project's GitHub Discussions!
