class Mdp < Formula
  desc "Paste clipboard image as Markdown link"
  homepage "https://github.com/daaa1k/mdp"
  version "0.2.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-macos-aarch64"
      sha256 "3e0a61c00690e06a42ff56648c9106f438942c5d8727fde65c336eeec9b314c5" # macos-aarch64

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
    sha256 "ff466ad301ad9df1dd7e085fcd3257a3c2a4c4667d3383b8c2568d332c7f81d0" # linux-x86_64

    def install
      bin.install "mdp-linux-x86_64" => "mdp"
    end
  end

  test do
    system "#{bin}/mdp", "--help"
  end
end
