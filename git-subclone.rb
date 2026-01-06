class GitSubclone < Formula
  desc "Manage nested git repositories with independent push capability"
  homepage "https://github.com/yejune/git-subclone"
  url "https://github.com/yejune/git-subclone/archive/refs/tags/v0.0.5.tar.gz"
  sha256 "c8d19ea956d3d9785de580dbf3d362e975631bdb413b1184bdf2e1c431209abc"
  license "MIT"
  head "https://github.com/yejune/git-subclone.git", branch: "main"

  depends_on "go" => :build

  def install
        ldflags = "-X github.com/yejune/git-subclone/cmd.Version=#{version}"
    system "go", "build", "-ldflags", ldflags, "-o", "git-subclone", "."
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
