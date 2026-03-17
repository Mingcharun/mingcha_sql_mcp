#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

// 获取平台和架构信息
const platform = process.platform;
const arch = process.arch;

console.log('🚀 Database MCP 安装中...');
console.log(`   平台: ${platform}`);
console.log(`   架构: ${arch}`);

// 映射到二进制文件名
function getBinaryName() {
  let platformName = '';
  let archName = '';
  let extension = '';

  // 平台映射
  switch (platform) {
    case 'darwin':
      platformName = 'darwin';
      break;
    case 'linux':
      platformName = 'linux';
      break;
    case 'win32':
      platformName = 'windows';
      extension = '.exe';
      break;
    default:
      throw new Error(`不支持的平台: ${platform}`);
  }

  // 架构映射
  switch (arch) {
    case 'x64':
      archName = 'amd64';
      break;
    case 'arm64':
      archName = 'arm64';
      break;
    default:
      throw new Error(`不支持的架构: ${arch}`);
  }

  return `database-mcp_${platformName}_${archName}${extension}`;
}

// 获取最新版本号
function getLatestVersion() {
  const packageJson = require('./package.json');
  return packageJson.version;
}

// 下载文件
function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    console.log(`📥 下载: ${url}`);

    const file = fs.createWriteStream(dest);

    https.get(url, (response) => {
      // 处理重定向
      if (response.statusCode === 302 || response.statusCode === 301) {
        return downloadFile(response.headers.location, dest)
          .then(resolve)
          .catch(reject);
      }

      if (response.statusCode !== 200) {
        reject(new Error(`下载失败，HTTP 状态码: ${response.statusCode}`));
        return;
      }

      const totalSize = parseInt(response.headers['content-length'], 10);
      let downloadedSize = 0;
      let lastPercent = 0;

      response.on('data', (chunk) => {
        downloadedSize += chunk.length;
        const percent = Math.floor((downloadedSize / totalSize) * 100);
        if (percent > lastPercent && percent % 10 === 0) {
          process.stdout.write(`\r   进度: ${percent}%`);
          lastPercent = percent;
        }
      });

      response.pipe(file);

      file.on('finish', () => {
        file.close();
        console.log('\n✓ 下载完成');
        resolve();
      });
    }).on('error', (err) => {
      fs.unlink(dest, () => {});
      reject(err);
    });
  });
}

// 主安装流程
async function install() {
  try {
    const binaryName = getBinaryName();
    const version = getLatestVersion();

    console.log(`   版本: v${version}`);
    console.log(`   二进制: ${binaryName}\n`);

    // 下载 URL
    const downloadUrl = `https://github.com/Mingcharun/mingcha_sql_mcp/releases/download/v${version}/${binaryName}`;

    // 目标路径
    const binDir = path.join(__dirname, 'bin');
    const targetName = platform === 'win32' ? 'database-mcp.exe' : 'database-mcp';
    const binaryPath = path.join(binDir, targetName);

    // 创建 bin 目录
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    // 下载二进制文件
    await downloadFile(downloadUrl, binaryPath);

    // 设置执行权限（Unix 系统）
    if (platform !== 'win32') {
      fs.chmodSync(binaryPath, 0o755);
      console.log('✓ 设置执行权限');
    }

    // 验证二进制文件
    console.log('🔍 验证二进制文件...');
    const stats = fs.statSync(binaryPath);
    console.log(`   大小: ${(stats.size / 1024 / 1024).toFixed(2)} MB`);

    console.log('\n✅ 安装成功！');
    console.log('\n💡 使用方法:');
    console.log('   npx @mingcharun/database-mcp');
    console.log('\n📖 更多信息:');
    console.log('   https://github.com/Mingcharun/mingcha_sql_mcp\n');

  } catch (error) {
    console.error('\n❌ 安装失败:', error.message);
    console.error('\n💡 你可以手动下载二进制文件:');
    console.error(`   https://github.com/Mingcharun/mingcha_sql_mcp/releases\n`);
    process.exit(1);
  }
}

// 运行安装
install();
