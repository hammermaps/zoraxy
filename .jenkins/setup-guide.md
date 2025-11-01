# Jenkins Setup Guide for Zoraxy

This comprehensive guide will walk you through setting up a complete Jenkins CI/CD pipeline for the Zoraxy project.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Jenkins Installation](#jenkins-installation)
3. [Plugin Installation](#plugin-installation)
4. [Jenkins Configuration](#jenkins-configuration)
5. [Pipeline Setup](#pipeline-setup)
6. [Advanced Configuration](#advanced-configuration)
7. [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

- **Operating System**: Linux (Ubuntu/Debian recommended), Windows, or macOS
- **Java**: JDK 11 or JDK 17
- **RAM**: Minimum 2GB, recommended 4GB+
- **Disk Space**: 10GB minimum for Jenkins + build artifacts

### Required Software

1. **Go**: Version 1.23 or higher
   ```bash
   # Ubuntu/Debian
   wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
   echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
   source ~/.bashrc
   ```

2. **Git**: Latest version
   ```bash
   sudo apt-get update
   sudo apt-get install git
   ```

3. **Docker** (optional, for Docker builds):
   ```bash
   curl -fsSL https://get.docker.com -o get-docker.sh
   sudo sh get-docker.sh
   sudo usermod -aG docker jenkins
   ```

## Jenkins Installation

### Ubuntu/Debian

```bash
# Add Jenkins repository
curl -fsSL https://pkg.jenkins.io/debian-stable/jenkins.io-2023.key | sudo tee \
  /usr/share/keyrings/jenkins-keyring.asc > /dev/null

echo deb [signed-by=/usr/share/keyrings/jenkins-keyring.asc] \
  https://pkg.jenkins.io/debian-stable binary/ | sudo tee \
  /etc/apt/sources.list.d/jenkins.list > /dev/null

# Update and install
sudo apt-get update
sudo apt-get install fontconfig openjdk-17-jre
sudo apt-get install jenkins

# Start Jenkins
sudo systemctl enable jenkins
sudo systemctl start jenkins
sudo systemctl status jenkins
```

### Using Docker

```bash
docker run -d \
  --name jenkins \
  -p 8080:8080 -p 50000:50000 \
  -v jenkins_home:/var/jenkins_home \
  -v /var/run/docker.sock:/var/run/docker.sock \
  jenkins/jenkins:lts
```

### Initial Setup

1. Access Jenkins at `http://localhost:8080`
2. Get the initial admin password:
   ```bash
   sudo cat /var/lib/jenkins/secrets/initialAdminPassword
   ```
3. Install suggested plugins
4. Create your first admin user

## Plugin Installation

### Required Plugins

Install these plugins via: **Manage Jenkins** → **Manage Plugins** → **Available**

1. **Pipeline** - Essential for pipeline support
2. **Git Plugin** - Git repository integration
3. **Pipeline: Stage View** - Visualize pipeline stages
4. **Email Extension Plugin** - Enhanced email notifications
5. **Workspace Cleanup Plugin** - Clean workspace after builds
6. **Credentials Plugin** - Manage credentials securely

### Optional but Recommended Plugins

7. **Docker Pipeline** - Docker integration
8. **Blue Ocean** - Modern UI for pipelines
9. **Parameterized Trigger** - Build parameters
10. **Build Monitor Plugin** - Dashboard for build status
11. **Pipeline Graph View** - Better pipeline visualization
12. **AnsiColor** - Colored console output
13. **Timestamper** - Add timestamps to console

### Installation via Jenkins CLI

```bash
# Download Jenkins CLI
wget http://localhost:8080/jnlpJars/jenkins-cli.jar

# Install plugins
java -jar jenkins-cli.jar -s http://localhost:8080/ install-plugin \
  workflow-aggregator \
  git \
  pipeline-stage-view \
  email-ext \
  ws-cleanup \
  credentials \
  docker-workflow

# Restart Jenkins
java -jar jenkins-cli.jar -s http://localhost:8080/ safe-restart
```

## Jenkins Configuration

### Global Tool Configuration

**Manage Jenkins** → **Global Tool Configuration**

#### Git Configuration

1. Name: `Default`
2. Path to Git executable: `git` (or full path)

#### Go Configuration

1. Click **Add Go**
2. Name: `Go 1.24`
3. Version: Select or specify 1.24.0
4. Or use custom installation path

### System Configuration

**Manage Jenkins** → **Configure System**

#### Email Notification

1. SMTP server: `smtp.gmail.com` (or your SMTP server)
2. Advanced:
   - Use SMTP Authentication: ✓
   - User Name: your-email@gmail.com
   - Password: your-app-password
   - Use SSL: ✓
   - SMTP Port: 465

#### Extended E-mail Notification

1. SMTP server: `smtp.gmail.com`
2. Default user E-mail suffix: `@yourdomain.com`
3. Advanced:
   - Use SMTP Authentication: ✓
   - User Name: your-email@gmail.com
   - Password: your-app-password
   - Use SSL: ✓
   - SMTP Port: 465

### Jenkins User Permissions

If Jenkins is running as a system service:

```bash
# Add jenkins user to docker group (if using Docker)
sudo usermod -aG docker jenkins

# Allow jenkins user to use sudo without password (optional, for specific commands)
sudo visudo
# Add: jenkins ALL=(ALL) NOPASSWD: /usr/local/bin/docker

# Restart Jenkins
sudo systemctl restart jenkins
```

## Pipeline Setup

### Method 1: Pipeline from SCM (Recommended)

1. **Create New Item**
   - Click **New Item**
   - Enter name: `Zoraxy-Build`
   - Select: **Pipeline**
   - Click **OK**

2. **General Configuration**
   - Description: `Automated build pipeline for Zoraxy`
   - ✓ Discard old builds
     - Days to keep builds: 30
     - Max # of builds to keep: 10

3. **Build Triggers** (Optional)
   - ✓ Poll SCM
   - Schedule: `H/15 * * * *` (every 15 minutes)
   
   Or for GitHub webhook:
   - ✓ GitHub hook trigger for GITScm polling

4. **Pipeline Configuration**
   - Definition: **Pipeline script from SCM**
   - SCM: **Git**
   - Repository URL: `https://github.com/yourusername/zoraxy.git`
   - Credentials: Add if private repository
   - Branch Specifier: `*/main` or `*/master`
   - Script Path: `Jenkinsfile`

5. **Save**

### Method 2: Inline Pipeline Script

1. Follow steps 1-3 above
2. **Pipeline Configuration**
   - Definition: **Pipeline script**
   - Copy and paste the content from `Jenkinsfile`
3. **Save**

### Setting Up Build Parameters

Edit your pipeline job:

1. **General** section
2. ✓ **This project is parameterized**
3. Add parameters matching those in the Jenkinsfile:
   - **Choice Parameter**: BUILD_TYPE (release, development)
   - **Boolean Parameter**: BUILD_DOCKER
   - **Boolean Parameter**: RUN_TESTS

## Advanced Configuration

### Setting Up Credentials

**Manage Jenkins** → **Manage Credentials**

#### Docker Registry Credentials

1. Click **Add Credentials**
2. Kind: **Username with password**
3. Scope: **Global**
4. Username: Your Docker Hub username
5. Password: Your Docker Hub password or access token
6. ID: `docker-registry`
7. Description: `Docker Hub credentials`

#### Git Credentials (if private repo)

1. Click **Add Credentials**
2. Kind: **Username with password** or **SSH Username with private key**
3. Configure accordingly

### Environment Variables

**Manage Jenkins** → **Configure System** → **Global properties**

✓ Environment variables

Add:
- `DOCKER_REGISTRY` = `docker.io/yourusername`
- `DEFAULT_RECIPIENTS` = `team@example.com`

### Webhook Setup (GitHub)

For automatic builds on push:

1. **GitHub Repository** → **Settings** → **Webhooks**
2. Click **Add webhook**
3. Payload URL: `http://your-jenkins-server:8080/github-webhook/`
4. Content type: `application/json`
5. Events: **Just the push event**
6. Active: ✓

### Multi-branch Pipeline

For more advanced setups with multiple branches:

1. **New Item** → **Multibranch Pipeline**
2. **Branch Sources** → **Add source** → **Git**
3. Project Repository: Your Zoraxy repository URL
4. **Build Configuration** → Script Path: `Jenkinsfile`
5. **Save**

### Parallel Builds with Agents

For faster builds across multiple machines:

```groovy
// In your Jenkinsfile
pipeline {
    agent none
    
    stages {
        stage('Build') {
            parallel {
                stage('Linux') {
                    agent { label 'linux' }
                    steps { /* build steps */ }
                }
                stage('Windows') {
                    agent { label 'windows' }
                    steps { /* build steps */ }
                }
            }
        }
    }
}
```

### Build Caching

Enable build caching for faster builds:

```groovy
// In Jenkinsfile
options {
    buildDiscarder(logRotator(numToKeepStr: '10'))
    // Enable workspace caching
    skipDefaultCheckout(true)
}

stages {
    stage('Checkout') {
        steps {
            checkout scm
        }
    }
}
```

## Monitoring and Maintenance

### Build Dashboard

Install **Build Monitor View** plugin:
1. **New View** → **Build Monitor View**
2. Select jobs to monitor
3. Configure refresh interval

### Disk Space Management

```bash
# Check Jenkins disk usage
du -sh /var/lib/jenkins

# Clean old workspaces
cd /var/lib/jenkins/workspace
find . -maxdepth 1 -type d -mtime +30 -exec rm -rf {} \;

# Clean old builds
# Configure in job: "Discard old builds"
```

### Backup Jenkins

```bash
# Backup Jenkins home
sudo tar -czf jenkins-backup-$(date +%Y%m%d).tar.gz /var/lib/jenkins/

# Or use ThinBackup plugin
```

## Troubleshooting

### Build Fails: "go: command not found"

**Solution**:
```bash
# Add Go to Jenkins PATH
# Manage Jenkins → Configure System → Global properties
# Environment variables
# PATH = /usr/local/go/bin:$PATH
```

### Build Fails: "Permission denied" for Docker

**Solution**:
```bash
sudo usermod -aG docker jenkins
sudo systemctl restart jenkins
```

### Workspace Issues

**Clear workspace**:
```bash
# In pipeline
stage('Clean') {
    steps {
        cleanWs()
    }
}
```

### Email Not Sending

**Check SMTP configuration**:
1. Use app-specific password (Gmail)
2. Enable "Less secure app access" or use OAuth
3. Test email:
   - **Manage Jenkins** → **Configure System** → **E-mail Notification**
   - Click **Test configuration**

### Pipeline Syntax Issues

Use Pipeline Syntax Generator:
- In your pipeline job → **Pipeline Syntax**
- Select step type
- Generate and copy syntax

### Build Hangs

Check:
1. Timeout settings
2. Interactive prompts in build scripts
3. Agent availability

## Testing Your Setup

### Manual Build

1. Go to your pipeline job
2. Click **Build with Parameters**
3. Select desired options
4. Click **Build**
5. Monitor console output

### Test Jenkinsfile Locally

```bash
# Install Jenkins Pipeline Linter
npm install -g jenkinsfile-runner

# Validate Jenkinsfile
jenkinsfile-runner validate Jenkinsfile
```

## Performance Optimization

### Parallel Stages

Already implemented in `Jenkinsfile.advanced` for multi-platform builds.

### Agent Labels

Configure agents with specific capabilities:
- `linux` - Linux build agents
- `docker` - Agents with Docker
- `go` - Agents with Go installed

### Build Queue Optimization

**Manage Jenkins** → **Configure System**
- Number of executors: Set based on CPU cores
- Quiet period: 5 seconds (reduce for faster builds)

## Security Best Practices

1. **Enable Security**: Manage Jenkins → Configure Global Security
2. **Use HTTPS**: Configure reverse proxy (Nginx/Apache)
3. **Regular Updates**: Keep Jenkins and plugins updated
4. **Secure Credentials**: Use Jenkins Credential Store
5. **Audit Logging**: Enable audit trail plugin
6. **Limited Permissions**: Configure matrix-based security

## Additional Resources

- [Jenkins Documentation](https://www.jenkins.io/doc/)
- [Pipeline Syntax Reference](https://www.jenkins.io/doc/book/pipeline/syntax/)
- [Jenkins Best Practices](https://www.jenkins.io/doc/book/pipeline/pipeline-best-practices/)
- [Zoraxy GitHub](https://github.com/tobychui/zoraxy)

## Support

For Jenkins-specific issues:
- [Jenkins Community](https://community.jenkins.io/)
- [Jenkins IRC](https://www.jenkins.io/chat/)

For Zoraxy build issues:
- [Zoraxy GitHub Issues](https://github.com/tobychui/zoraxy/issues)

---

**Happy Building! 🚀**
