class Mdp < Formula
  desc "Paste clipboard image as Markdown link"
  homepage "https://github.com/daaa1k/mdp"
  version "0.2.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-macos-aarch64"
      sha256 "5773dec21988dd34803acecd4adb303d2d43d0423fe30452eb8aba08e8c5109e" # macos-aarch64

      def install
        bin.install "mdp-macos-aarch64" => "mdp"
      end
    else
      url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-macos-x86_64"
      sha256 "" # macos-x86_64

      def install
        bin.install "mdp-macos-x86_64" => "mdp"
      end
    end
  end

  on_linux do
    url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-linux-x86_64"
    sha256 "e4d05ecca58460eb96b81ad9f514aa4c75b7a094abcf2645fd0c0b90b427dff1" # linux-x86_64

    def install
      bin.install "mdp-linux-x86_64" => "mdp"
    end
  end

  test do
    system "#{bin}/mdp", "--help"
  end
end
