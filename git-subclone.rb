class GitSubclone < Formula
  desc "Manage nested git repositories with independent push capability"
  homepage "https://github.com/yejune/git-subclone"
  url "https://github.com/yejune/git-subclone/archive/refs/tags/v0.0.11.tar.gz"
  sha256 "8da4925ca6d414717a03b9c8041f59b41ce070ad3ed013168445d052f7303b85"
  license "MIT"
  head "https://github.com/yejune/git-subclone.git", branch: "main"

  depends_on "go" => :build

  def install
        system "go", "build", "-o", "git-subclone", "./cmd/git-subclone"
    bin.install "git-subclone"
  end

  def test
        assert_match "version", shell_output("#{bin}/git-subclone --version")
  end

  def caveats
    <<~EOS
            git-subclone has been installed!
      
      Quick Start:
        git subclone https://github.com/user/repo.git
        git subclone https://github.com/user/repo.git packages/repo
        git subclone -b develop https://github.com/user/repo.git
      
      Commands:
        git subclone sync      Sync all subclones
        git subclone list      List subclones
        git subclone push      Push subclone(s)
        git subclone status    Show status
    EOS
  end
end
