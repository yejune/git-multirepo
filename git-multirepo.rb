class GitMultirepo < Formula
  desc "Multi-repository management tool for nested git repositories with independent push capability"
  homepage "https://github.com/yejune/git-multirepo"
  url "https://github.com/yejune/git-multirepo/archive/refs/tags/v0.2.31.tar.gz"
  sha256 "aea7c71b028cd4bf3a622d5a3d5434d2e8ed9d491bd9e8a95c83c7813d0fd547"
  license "MIT"
  head "https://github.com/yejune/git-multirepo.git", branch: "main"

  depends_on "go" => :build

  def install
        ldflags = "-X github.com/yejune/git-multirepo/cmd.Version=#{version}"
    system "go", "build", "-ldflags", ldflags, "-o", "git-multirepo", "."
    bin.install "git-multirepo"
  end

  def test
        assert_match "version", shell_output("#{bin}/git-multirepo --version")
  end

  def caveats
    <<~EOS
            git-multirepo has been installed!
      
      Quick Start:
        git multirepo clone https://github.com/user/repo.git
        git multirepo clone https://github.com/user/repo.git packages/repo
        git multirepo clone -b develop https://github.com/user/repo.git
      
      Commands:
        git multirepo sync      Apply configuration
        git multirepo list      List workspaces
        git multirepo status    Show status
        git multirepo reset     Reset workspaces
        git multirepo pull      Pull updates
    EOS
  end
end
