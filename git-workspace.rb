class GitWorkspace < Formula
  desc "Manage nested git repositories with independent push capability"
  homepage "https://github.com/yejune/git-workspace"
  url "https://github.com/yejune/git-workspace/archive/refs/tags/v0.2.14.tar.gz"
  sha256 "940e845550c88df71bd8da0280296ef21d6e84f10e7461c67172b5c782ff4fcd"
  license "MIT"
  head "https://github.com/yejune/git-workspace.git", branch: "main"

  depends_on "go" => :build

  def install
        ldflags = "-X github.com/yejune/git-workspace/cmd.Version=#{version}"
    system "go", "build", "-ldflags", ldflags, "-o", "git-workspace", "."
    bin.install "git-workspace"
  end

  def test
        assert_match "version", shell_output("#{bin}/git-workspace --version")
  end

  def caveats
    <<~EOS
            git-workspace has been installed!
      
      Quick Start:
        git workspace clone https://github.com/user/repo.git
        git workspace clone https://github.com/user/repo.git packages/repo
        git workspace clone -b develop https://github.com/user/repo.git
      
      Commands:
        git workspace sync      Apply configuration
        git workspace list      List workspaces
        git workspace status    Show status
        git workspace reset     Reset workspaces
        git workspace pull      Pull updates
    EOS
  end
end
