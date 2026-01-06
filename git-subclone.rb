class GitSubclone < Formula
  desc "Manage nested git repositories with dual tracking - files in both mother and sub repos"
  homepage "https://github.com/yejune/git-subclone"
  url "https://github.com/yejune/git-subclone/archive/refs/tags/v0.0.1.tar.gz"
  sha256 "211c210afc3f184284ca4e6d83a3d1a753dc66ec9de00d714e19e3603eb70353"
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
      
      Usage:
        git subclone add <repo> <path>   Add a subclone
        git subclone sync                Sync all subclones
        git subclone list                List subclones
        git subclone push [path]         Push subclone(s)
    EOS
  end
end
